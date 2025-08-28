/*
@Author: Lzww
@LastEditTime: 2025-8-28 20:56:43
@Description: Auto-tuning mechanism for SafeUDP protocol performance optimization
@Language: Go 1.23.4
*/

package safeudp

import (
	"container/heap"
	"runtime"
	"sync"
	"time"
)

var SystemTimer *Timer = NewTimer(runtime.NumCPU())

type timedFunc struct {
	execute func()
	ts      time.Time
}

type Timer struct {
	prependTasks    []timedFunc
	prependLock     sync.Mutex
	chPrependNotify chan any

	chTask chan timedFunc

	closeOnce sync.Once
	close     chan any
}

func NewTimer(parallel int) *Timer {
	t := new(Timer)
	t.chTask = make(chan timedFunc)
	t.close = make(chan any)
	t.chPrependNotify = make(chan any, 1)

	for i := 0; i < parallel; i++ {
		go t.seched()
	}

	go t.prepend()
	return t
}

type timeFuncHeap []timedFunc

func (h timeFuncHeap) Len() int {
	return len(h)
}

func (h timeFuncHeap) Less(i, j int) bool {
	return h[i].ts.Before(h[j].ts)
}

func (h timeFuncHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *timeFuncHeap) Push(x any) {
	*h = append(*h, x.(timedFunc))
}

func (h *timeFuncHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (t *Timer) seched() {
	timer := time.NewTimer(0)
	defer timer.Stop()

	var tasks timeFuncHeap
	drained := false

	for {
		select {
		case task := <-t.chTask:
			now := time.Now()
			if now.After(task.ts) {
				go task.execute()
			} else {
				heap.Push(&tasks, task)
				stopped := timer.Stop()
				if !stopped && !drained {
					<-timer.C
				}
				if tasks.Len() > 0 {
					timer.Reset(tasks[0].ts.Sub(now))
				}
			}
		case now := <-timer.C:
			drained = true
			for tasks.Len() > 0 {
				if now.After(tasks[0].ts) {
					task := heap.Pop(&tasks).(timedFunc)
					go task.execute()
				} else {
					timer.Reset(tasks[0].ts.Sub(now))
					drained = false
					break
				}
			}
		case <-t.close:
			return
		}

	}
}

func (t *Timer) prepend() {
	var tasks []timedFunc
	for {
		select {
		case <-t.chPrependNotify:
			t.prependLock.Lock()
			if cap(tasks) < cap(t.prependTasks) {
				tasks = make([]timedFunc, 0, cap(t.prependTasks))
			}
			tasks = tasks[:len(t.prependTasks)]
			copy(tasks, t.prependTasks)
			for k := range t.prependTasks {
				t.prependTasks[k].execute = nil
			}
			t.prependTasks = t.prependTasks[:0]
			t.prependLock.Unlock()

			for k := range tasks {
				select {
				case t.chTask <- tasks[k]:
					tasks[k].execute = nil
				case <-t.close:
					return
				}
			}
			tasks = tasks[:0]
		case <-t.close:
			return
		}
	}

}

func (t *Timer) Put(f func(), deadline time.Time) {
	t.prependLock.Lock()
	t.prependTasks = append(t.prependTasks, timedFunc{f, deadline})
	t.prependLock.Unlock()

	select {
	case t.chPrependNotify <- struct{}{}:
	default:
	}
}

func (t *Timer) Close() {
	t.closeOnce.Do(func() {
		close(t.close)
	})
}

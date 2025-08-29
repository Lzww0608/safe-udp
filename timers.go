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

// SystemTimer is a global timer instance initialized with the number of CPU cores
// It provides a shared timer service for the entire SafeUDP package
var SystemTimer *Timer = NewTimer(runtime.NumCPU())

// timedFunc represents a function that should be executed at a specific time
type timedFunc struct {
	execute func()    // The function to execute
	ts      time.Time // The timestamp when the function should be executed
}

// Timer manages scheduled function execution with multiple worker goroutines
// It uses a heap-based priority queue to efficiently handle timed tasks
type Timer struct {
	prependTasks    []timedFunc // Buffer for new tasks before they're processed
	prependLock     sync.Mutex  // Mutex to protect prependTasks
	chPrependNotify chan any    // Channel to notify when new tasks are added

	chTask chan timedFunc // Channel to send tasks to worker goroutines

	closeOnce sync.Once // Ensures Close() is called only once
	close     chan any  // Channel to signal shutdown to all goroutines
}

// NewTimer creates a new Timer with the specified number of parallel worker goroutines
func NewTimer(parallel int) *Timer {
	t := new(Timer)
	t.chTask = make(chan timedFunc)
	t.close = make(chan any)
	t.chPrependNotify = make(chan any, 1)

	// Start worker goroutines for task scheduling
	for i := 0; i < parallel; i++ {
		go t.seched()
	}

	// Start the prepend goroutine to handle new task additions
	go t.prepend()
	return t
}

// timeFuncHeap implements heap.Interface for timedFunc elements
// It creates a min-heap ordered by execution time
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

// seched is the main scheduling loop for each worker goroutine
// It manages a heap of pending tasks and executes them at the right time
func (t *Timer) seched() {
	timer := time.NewTimer(0)
	defer timer.Stop()

	var tasks timeFuncHeap // Min-heap of pending tasks
	drained := false       // Flag to track if timer channel was drained

	for {
		select {
		case task := <-t.chTask:
			now := time.Now()
			// If the task should be executed immediately, run it
			if now.After(task.ts) {
				go task.execute()
			} else {
				// Add task to heap and reset timer for the earliest task
				heap.Push(&tasks, task)
				stopped := timer.Stop()
				if !stopped && !drained {
					<-timer.C // Drain the timer channel if it wasn't stopped
				}
				if tasks.Len() > 0 {
					timer.Reset(tasks[0].ts.Sub(now))
				}
			}
		case now := <-timer.C:
			drained = true
			// Execute all tasks that are due
			for tasks.Len() > 0 {
				if now.After(tasks[0].ts) {
					task := heap.Pop(&tasks).(timedFunc)
					go task.execute()
				} else {
					// Reset timer for the next task and break
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

// prepend handles the addition of new tasks to the timer
// It runs in a separate goroutine to avoid blocking the main scheduling loops
func (t *Timer) prepend() {
	var tasks []timedFunc
	for {
		select {
		case <-t.chPrependNotify:
			// Lock and copy all pending tasks
			t.prependLock.Lock()
			if cap(tasks) < cap(t.prependTasks) {
				tasks = make([]timedFunc, 0, cap(t.prependTasks))
			}
			tasks = tasks[:len(t.prependTasks)]
			copy(tasks, t.prependTasks)
			// Clear function references to prevent memory leaks
			for k := range t.prependTasks {
				t.prependTasks[k].execute = nil
			}
			t.prependTasks = t.prependTasks[:0]
			t.prependLock.Unlock()

			// Send all tasks to worker goroutines
			for k := range tasks {
				select {
				case t.chTask <- tasks[k]:
					tasks[k].execute = nil // Clear reference after sending
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

// Put adds a new function to be executed at the specified deadline
// The function will be executed by one of the worker goroutines
func (t *Timer) Put(f func(), deadline time.Time) {
	t.prependLock.Lock()
	t.prependTasks = append(t.prependTasks, timedFunc{f, deadline})
	t.prependLock.Unlock()

	// Notify the prepend goroutine that new tasks are available
	select {
	case t.chPrependNotify <- struct{}{}:
	default: // Don't block if notification is already pending
	}
}

// Close shuts down the timer and all its worker goroutines
// It can be called multiple times safely
func (t *Timer) Close() {
	t.closeOnce.Do(func() {
		close(t.close)
	})
}

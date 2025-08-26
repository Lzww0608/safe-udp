/*
@Author: Lzww
@LastEditTime: 2025-8-26 19:53:13
@Description: Ring Buffer
@Language: Go 1.23.4
*/

package safeudp

import (
	"testing"
)

func TestRingBuffer_BasicOperations(t *testing.T) {
	rb := &RingBuffer[int]{
		buffer: make([]int, 5),
	}

	// 测试空缓冲区
	if !rb.Empty() {
		t.Error("新创建的环形缓冲区应该为空")
	}

	if rb.Len() != 0 {
		t.Errorf("空缓冲区长度应该为0，实际为%d", rb.Len())
	}

	// 测试Push操作
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	if rb.Empty() {
		t.Error("添加元素后缓冲区不应该为空")
	}

	if rb.Len() != 3 {
		t.Errorf("缓冲区长度应该为3，实际为%d", rb.Len())
	}

	// 测试Pop操作
	val, ok := rb.Pop()
	if !ok || val != 1 {
		t.Errorf("Pop应该返回1，实际返回%d", val)
	}

	if rb.Len() != 2 {
		t.Errorf("Pop后缓冲区长度应该为2，实际为%d", rb.Len())
	}

	// 测试Peek操作
	peekVal, ok := rb.Peek()
	if !ok || *peekVal != 2 {
		t.Errorf("Peek应该返回2，实际返回%d", *peekVal)
	}

	if rb.Len() != 2 {
		t.Errorf("Peek后缓冲区长度应该保持为2，实际为%d", rb.Len())
	}
}

func TestRingBuffer_FullAndGrow(t *testing.T) {
	rb := &RingBuffer[int]{
		buffer: make([]int, 3), // 只能存储2个元素（保留一个空位）
	}

	// 填满缓冲区
	rb.Push(1)
	rb.Push(2)

	if !rb.Full() {
		t.Error("缓冲区应该已满")
	}

	if rb.MaxLen() != 2 {
		t.Errorf("最大长度应该为2，实际为%d", rb.MaxLen())
	}

	// 测试自动扩容
	rb.Push(3) // 这应该触发扩容

	if rb.Full() {
		t.Error("扩容后缓冲区不应该满")
	}

	if rb.Len() != 3 {
		t.Errorf("扩容后长度应该为3，实际为%d", rb.Len())
	}
}

func TestRingBuffer_EmptyOperations(t *testing.T) {
	rb := &RingBuffer[int]{
		buffer: make([]int, 5),
	}

	// 测试空缓冲区的Pop操作
	val, ok := rb.Pop()
	if ok {
		t.Error("空缓冲区Pop应该返回false")
	}
	if val != 0 {
		t.Errorf("空缓冲区Pop应该返回零值，实际返回%d", val)
	}

	// 测试空缓冲区的Peek操作
	peekVal, ok := rb.Peek()
	if ok {
		t.Error("空缓冲区Peek应该返回false")
	}
	if peekVal != nil {
		t.Error("空缓冲区Peek应该返回nil")
	}
}

func TestRingBuffer_ForEach(t *testing.T) {
	rb := &RingBuffer[int]{
		buffer: make([]int, 10),
	}

	// 添加一些元素
	for i := 1; i <= 5; i++ {
		rb.Push(i)
	}

	// 测试ForEach
	var result []int
	rb.ForEach(func(val *int) bool {
		result = append(result, *val)
		return true
	})

	expected := []int{1, 2, 3, 4, 5}
	if len(result) != len(expected) {
		t.Errorf("ForEach结果长度不匹配，期望%d，实际%d", len(expected), len(result))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("ForEach结果不匹配，位置%d期望%d，实际%d", i, expected[i], v)
		}
	}

	// 测试提前停止
	var partialResult []int
	rb.ForEach(func(val *int) bool {
		partialResult = append(partialResult, *val)
		return *val < 3 // 在值为3时停止
	})

	if len(partialResult) != 3 {
		t.Errorf("提前停止的ForEach应该返回3个元素，实际返回%d个", len(partialResult))
	}
}

func TestRingBuffer_ForEachReverse(t *testing.T) {
	rb := &RingBuffer[int]{
		buffer: make([]int, 10),
	}

	// 添加一些元素
	for i := 1; i <= 5; i++ {
		rb.Push(i)
	}

	// 测试ForEachReverse
	var result []int
	rb.ForEachReverse(func(val *int) bool {
		result = append(result, *val)
		return true
	})

	expected := []int{5, 4, 3, 2, 1}
	if len(result) != len(expected) {
		t.Errorf("ForEachReverse结果长度不匹配，期望%d，实际%d", len(expected), len(result))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("ForEachReverse结果不匹配，位置%d期望%d，实际%d", i, expected[i], v)
		}
	}
}

func TestRingBuffer_Discard(t *testing.T) {
	rb := &RingBuffer[int]{
		buffer: make([]int, 10),
	}

	// 添加一些元素
	for i := 1; i <= 5; i++ {
		rb.Push(i)
	}

	// 测试丢弃部分元素
	discarded := rb.Discard(2)
	if discarded != 2 {
		t.Errorf("应该丢弃2个元素，实际丢弃%d个", discarded)
	}

	if rb.Len() != 3 {
		t.Errorf("丢弃后长度应该为3，实际为%d", rb.Len())
	}

	// 验证剩余元素
	val, ok := rb.Pop()
	if !ok || val != 3 {
		t.Errorf("丢弃后第一个元素应该为3，实际为%d", val)
	}

	// 测试丢弃超过现有元素数量
	discarded = rb.Discard(10)
	if discarded != 2 {
		t.Errorf("应该只能丢弃2个元素，实际丢弃%d个", discarded)
	}

	if !rb.Empty() {
		t.Error("丢弃所有元素后缓冲区应该为空")
	}

	// 测试在空缓冲区上丢弃
	discarded = rb.Discard(5)
	if discarded != 0 {
		t.Errorf("空缓冲区丢弃应该返回0，实际返回%d", discarded)
	}
}

func TestRingBuffer_Wraparound(t *testing.T) {
	rb := &RingBuffer[int]{
		buffer: make([]int, 5),
	}

	// 填充缓冲区
	for i := 1; i <= 4; i++ {
		rb.Push(i)
	}

	// 弹出一些元素
	rb.Pop()
	rb.Pop()

	// 再添加元素，这会导致环绕
	rb.Push(5)
	rb.Push(6)
	rb.Push(7)

	// 验证顺序
	expected := []int{3, 4, 5, 6, 7}
	for _, exp := range expected {
		val, ok := rb.Pop()
		if !ok || val != exp {
			t.Errorf("环绕测试失败，期望%d，实际%d", exp, val)
		}
	}
}

func TestRingBuffer_StringType(t *testing.T) {
	rb := &RingBuffer[string]{
		buffer: make([]string, 5),
	}

	// 测试字符串类型
	rb.Push("hello")
	rb.Push("world")

	val, ok := rb.Pop()
	if !ok || val != "hello" {
		t.Errorf("字符串测试失败，期望'hello'，实际'%s'", val)
	}

	peekVal, ok := rb.Peek()
	if !ok || *peekVal != "world" {
		t.Errorf("字符串Peek测试失败，期望'world'，实际'%s'", *peekVal)
	}
}

package util

// Stack 泛型栈，底层使用切片
type Stack[T any] struct {
	items []T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

// Push 将元素压入栈顶
func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

// Pop 弹出栈顶元素，返回元素和是否成功（栈为空时返回零值和 false）
func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, true
}

// Peek 查看栈顶元素但不弹出，返回元素和是否成功
func (s *Stack[T]) Peek() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

// Len 返回栈中元素个数
func (s *Stack[T]) Len() int {
	return len(s.items)
}

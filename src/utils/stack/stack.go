package stack

import "container/vector"

type Stack vector.Vector

func (s *Stack) Push(value interface{}) {
	if value != nil {
		s.Push(value)
	}
	return
}

func (s *Stack) Pop() interface{} {

	return s.Pop()
}

func (s *Stack) Len() int {

	return s.Len()
}

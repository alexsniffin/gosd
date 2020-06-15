package gopd

import (
	"container/heap"
	"sync"
)

type SynchronousHeap struct {
	pq priorityQueue

	mux sync.Mutex
}

func NewSynchronousHeap(pq priorityQueue) SynchronousHeap {
	heap.Init(&pq)
	return SynchronousHeap{
		pq: pq,

		mux: sync.Mutex{},
	}
}

func (s *SynchronousHeap) Len() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.pq.Len()
}

func (s *SynchronousHeap) Pop() interface{} {
	s.mux.Lock()
	defer s.mux.Unlock()
	return heap.Pop(&s.pq)
}

func (s *SynchronousHeap) Push(msg interface{}) {
	s.mux.Lock()
	defer s.mux.Unlock()
	heap.Push(&s.pq, msg)
}

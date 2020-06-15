package gopd

type priorityQueue []*item

func (pq priorityQueue) Len() int {
	return len(pq)
}

func (pq priorityQueue) Less(i, j int) bool {
	if pq[i] == nil || pq[j] == nil {
		return true
	}
	return pq[i].Message.At.Before(pq[j].Message.At)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	if pq[i] != nil {
		pq[i].Index = i
	}
	if pq[j] != nil {
		pq[j].Index = j
	}
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := item{
		Message: x.(ScheduledMessage), // todo
		Index:   n,
	}
	*pq = append(*pq, &item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	if item != nil {
		item.Index = -1 // for safety
		*pq = old[0 : n-1]
		return item.Message
	}
	return nil
}

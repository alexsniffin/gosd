package gosd

type item struct {
	Message *ScheduledMessage
	Index   int
}

type priorityQueue struct {
	items         []*item
	maintainOrder bool
}

func (pq priorityQueue) Len() int {
	return len(pq.items)
}

func (pq priorityQueue) Less(i, j int) bool {
	return pq.items[i].Message.At.Before(pq.items[j].Message.At)
}

func (pq priorityQueue) Swap(i, j int) {
	// if items have the same time, don't swap
	if pq.items[i].Message.At.Equal(pq.items[j].Message.At) {
		return
	}
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].Index = i
	pq.items[j].Index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(pq.items)
	item := item{
		Message: x.(*ScheduledMessage),
		Index:   n,
	}
	pq.items = append(pq.items, &item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old.items)
	itm := old.items[n-1]

	// will check equality of dispatch time up to the N-th item and dispatch the items in fifo which changes Pop
	// worst-case complexity to O(nlogn), i.e. all items have the same dispatch time
	if pq.maintainOrder {
		i := 2
		var nextItem *item
		if n >= i {
			if old.items[n-i].Message.At.Equal(itm.Message.At) {
				nextItem = old.items[n-i]
				for i <= n && nextItem.Message.At.Equal(itm.Message.At) {
					nextItem = old.items[n-i]
					i++
				}
			}
		}
		if nextItem != nil {
			old.items[n-(i-1)] = nil
			nextItem.Index = -1
			var newPq []*item
			if n-(i-1) == 0 {
				newPq = old.items[1:n]
			} else {
				newPq = append(old.items[0:n-(i-1)], append(old.items[n-(i-2):n])...)
			}
			pq.items = newPq
			return nextItem.Message
		}
	}

	old.items[n-1] = nil // avoid memory leak
	itm.Index = -1       // for safety
	pq.items = old.items[0 : n-1]
	return itm.Message
}

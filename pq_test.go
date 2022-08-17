package gosd

import (
	"testing"
	"time"
)

func TestPriorityQueue_Len(t *testing.T) {
	t.Parallel()

	pq := priorityQueue[any]{
		items: []*item[any]{{
			Message: &ScheduledMessage[any]{},
			Index:   0,
		}},
		maintainOrder: false,
	}

	result := pq.Len()

	if result != 1 {
		t.Errorf("Len() unexpected value = %d, want 1", result)
	}
}

func TestPriorityQueue_Less(t *testing.T) {
	pq := priorityQueue[any]{
		items: []*item[any]{
			{
				Message: &ScheduledMessage[any]{
					At: time.Now(),
				},
				Index: 0,
			},
			{
				Message: &ScheduledMessage[any]{
					At: time.Now().Add(1 * time.Second),
				},
				Index: 1,
			}},
		maintainOrder: false,
	}

	result := pq.Less(0, 1)

	if !result {
		t.Errorf("Len() unexpected value = %v, want true", result)
	}
}

func TestPriorityQueue_Swap(t *testing.T) {
	t.Parallel()

	t.Run("notEqual", func(t *testing.T) {
		pq := priorityQueue[any]{
			items: []*item[any]{
				{
					Message: &ScheduledMessage[any]{
						At: time.Now(),
					},
					Index: 0,
				},
				{
					Message: &ScheduledMessage[any]{
						At: time.Now().Add(1 * time.Second),
					},
					Index: 1,
				}},
			maintainOrder: false,
		}
		item1 := pq.items[0]
		item2 := pq.items[1]

		pq.Swap(0, 1)

		if pq.items[0] != item2 && pq.items[1] != item1 {
			t.Errorf("Swap() unexpected items = %v", pq.items)
		}
	})

	t.Run("equal", func(t *testing.T) {
		at := time.Now()
		pq := priorityQueue[any]{
			items: []*item[any]{
				{
					Message: &ScheduledMessage[any]{
						At: at,
					},
					Index: 0,
				},
				{
					Message: &ScheduledMessage[any]{
						At: at,
					},
					Index: 1,
				}},
			maintainOrder: false,
		}
		item1 := pq.items[0]
		item2 := pq.items[1]

		pq.Swap(0, 1)

		if pq.items[0] != item1 && pq.items[1] != item2 {
			t.Errorf("Swap() unexpected items = %v", pq.items)
		}
	})
}

func TestPriorityQueue_Push(t *testing.T) {
	t.Parallel()

	pq := priorityQueue[any]{
		maintainOrder: false,
	}

	pq.Push(&ScheduledMessage[any]{})

	if len(pq.items) != 1 {
		t.Errorf("Push() unexpect length = %d, want 1", len(pq.items))
	}
}

func TestPriorityQueue_Pop(t *testing.T) {
	t.Parallel()

	t.Run("unequal", func(t *testing.T) {
		pq := priorityQueue[any]{
			items: []*item[any]{
				{
					Message: &ScheduledMessage[any]{
						At:      time.Now(),
						Message: 0,
					},
					Index: 0,
				},
				{
					Message: &ScheduledMessage[any]{
						At:      time.Now().Add(1 * time.Second),
						Message: 1,
					},
					Index: 1,
				}},
			maintainOrder: false,
		}

		result := pq.Pop()

		message, ok := result.(*ScheduledMessage[any])
		if !ok {
			t.Error("Pop() unexpected type")
			t.FailNow()
		}

		if message.Message.(int) != 1 {
			t.Errorf("Pop() unexpected result = %d, want 1", message.Message.(int))
		}
	})

	t.Run("equalMaintainOrder", func(t *testing.T) {
		at := time.Now()
		pq := priorityQueue[any]{
			items: []*item[any]{
				{
					Message: &ScheduledMessage[any]{
						At:      at,
						Message: 0,
					},
					Index: 0,
				},
				{
					Message: &ScheduledMessage[any]{
						At:      at,
						Message: 1,
					},
					Index: 1,
				},
				{
					Message: &ScheduledMessage[any]{
						At:      at,
						Message: 2,
					},
					Index: 2,
				},
			},
			maintainOrder: true,
		}

		result := pq.Pop()

		message, ok := result.(*ScheduledMessage[any])
		if !ok {
			t.Error("Pop() unexpected type")
			t.FailNow()
		}

		if message.Message.(int) != 0 {
			t.Errorf("Pop() unexpected result = %d, want 0", message.Message.(int))
		}
	})
}

package list

import (
	"sync"
	"testing"
)

func TestLRUListNode_Next(t *testing.T) {
	l := NewLRUList[int, int]()

	node1 := l.PushFront(1, 1)
	node2 := l.PushFront(2, 2)

	next := node1.Next()
	if next != node2 {
		t.Errorf("error: node2 should be next after node1")
	}

	next = next.Next()
	if next != nil {
		t.Errorf("error: expected nil")
	}
}

func TestLRUListNode_Prev(t *testing.T) {
	l := NewLRUList[int, int]()

	node1 := l.PushFront(1, 1)
	node2 := l.PushFront(2, 2)

	prev := node2.Prev()
	if prev != node1 {
		t.Errorf("error: node1 should be previous to node2")
	}

	prev = prev.Prev()
	if prev != nil {
		t.Errorf("error: expected nil")
	}
}

func TestLRUList_Front(t *testing.T) {
	l := NewLRUList[int, int]()

	node1 := l.PushFront(2, 2)
	l.PushBack(1, 1)

	if l.Front() != node1 {
		t.Errorf("error: expected node (2, 2)")
	}
	node2 := l.PushFront(3, 3)
	if l.Front() != node2 {
		t.Errorf("error: expected node (3, 3)")
	}
	_, err := l.PopFront()
	if err != nil {
		t.Fatalf("error %v", err)
	}
	if l.Front() != node1 {
		t.Errorf("error: expected node (2, 2)")
	}
}

func TestLRUList_Back(t *testing.T) {
	l := NewLRUList[int, int]()

	l.PushFront(2, 2)
	node2 := l.PushBack(1, 1)

	if l.Back() != node2 {
		t.Errorf("error: expected node (1, 1)")
	}
	node3 := l.PushFront(3, 3)
	if l.Front() != node3 {
		t.Errorf("error: expected node (3, 3)")
	}
	_, err := l.PopBack()
	if err != nil {
		t.Fatalf("error %v", err)
	}
	if l.Front() != node3 {
		t.Errorf("error: expected node (3, 3)")
	}

}

func TestLRUList_MoveToFront(t *testing.T) {
	l := NewLRUList[int, int]()

	node1 := l.PushFront(1, 1)
	l.PushFront(2, 2)
	l.PushFront(3, 3)

	l.MoveToFront(node1)

}

func TestLRUList_MoveToBack(t *testing.T) {

}

func TestLRUList_Size(t *testing.T) {
	l := NewLRUList[int, int]()

	node1 := l.PushFront(1, 1)
	l.PushFront(2, 2)

	if l.Size() != 2 {
		t.Errorf("error: size should be 2")
	}

	_, err := l.Remove(node1)

	if err != nil {
		t.Fatalf("error %v", err)
	}

	if l.Size() != 1 {
		t.Errorf("error: size should be 1")
	}
}

func TestLRUList_Insert(t *testing.T) {
	l := NewLRUList[int, int]()

	node1 := l.PushFront(1, 1)
	if node1.Key != 1 || node1.Value != 1 {
		t.Fatalf("error: node1 added incorrectly")
	}
	node2 := l.PushFront(2, 2)
	if node2.Key != 2 || node2.Value != 2 {
		t.Fatalf("error: node2 added incorrectly")
	}

	node3, err := l.Insert(3, 3, node1)
	if err != nil {
		t.Errorf("error: %v", err)
	}

	if node1.Next() != node3 || node2.Prev() != node3 {
		t.Errorf("error: node added incorrectly")
	}
}

func TestLRUList_PushFront(t *testing.T) {
	l := NewLRUList[int, int]()

	node1 := l.PushFront(1, 1)
	if node1.Key != 1 || node1.Value != 1 {
		t.Errorf("error: node1 added incorrectly")
	}
	node2 := l.PushFront(2, 2)
	if node2.Key != 2 || node2.Value != 2 {
		t.Errorf("error: node2 added incorrectly")
	}
	node3 := l.PushFront(3, 3)
	if l.Front() != node3 {
		t.Errorf("error: front should be node3")
	}
	if l.Back() != node1 {
		t.Errorf("error: front should be node3")
	}
}

func TestLRUList_PushBack(t *testing.T) {
	l := NewLRUList[int, int]()

	node1 := l.PushBack(1, 1)
	if node1.Key != 1 || node1.Value != 1 {
		t.Errorf("error: node1 added incorrectly")
	}
	node2 := l.PushBack(2, 2)
	if node2.Key != 2 || node2.Value != 2 {
		t.Errorf("error: node2 added incorrectly")
	}
	node3 := l.PushBack(3, 3)
	if l.Front() != node1 {
		t.Errorf("error: front should be node1")
	}
	if l.Back() != node3 {
		t.Errorf("error: front should be node3")
	}
}

func TestLRUList_Remove(t *testing.T) {
	l := NewLRUList[int, int]()

	l.PushFront(1, 1)
	node2 := l.PushFront(2, 2)
	l.PushFront(2, 2)
	node4, err := l.Remove(node2)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if node2 != node4 {
		t.Errorf("error: deleted node is incorrect")
	}
}

func TestLRUList_PopFront(t *testing.T) {
	l := NewLRUList[int, int]()

	l.PushFront(1, 1)
	node2 := l.PushFront(2, 2)
	node3 := l.PushFront(3, 3)
	node4, err := l.PopFront()
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if node4 != node3 {
		t.Errorf("error: deleted node is incorrect")
	}
	if node2.Next() != nil {
		t.Errorf("error: node is improperly deleted")
	}
}

func TestLRUList_PopBack(t *testing.T) {
	l := NewLRUList[int, int]()

	l.PushBack(1, 1)
	node2 := l.PushBack(2, 2)
	node3 := l.PushBack(3, 3)
	node4, err := l.PopBack()
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if node4 != node3 {
		t.Errorf("error: deleted node is incorrect")
	}
	if node2.Prev() != nil {
		t.Errorf("error: node is improperly deleted")
	}
}

func TestLRUList_Concurrency(t *testing.T) {
	l := NewLRUList[float64, int]()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for i := 0; i < 100000; i++ {
			l.PushFront(float64(i), i)
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 100000; i++ {
			l.Front()
		}
		wg.Done()
	}()

	wg.Wait()
	if l.Size() != 100000 {
		t.Errorf("%d", l.Size())
		t.Fail()
	}
}
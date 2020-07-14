package gonblocking

import (
	"sync/atomic"
	"unsafe"
)

const (
	FlagOk uint32 = 0
	FlagDeleted uint32 = 1
)
type Queue interface {
	Len() int
	Push(interface{}) bool
	Pop() (interface{}, bool)
	Peek() (interface{}, bool)
}

type Element struct {
	val  interface{}
	next unsafe.Pointer
	flag uint32
}

func newElement(val interface{}) *Element {
	return &Element{
		val:  val,
		next: nil,
	}
}

func (e *Element) setFlag(flag uint32) {
	atomic.StoreUint32(&e.flag, flag)
}
func (e *Element) getFlag() uint32 {
	return atomic.LoadUint32(&e.flag)
}
func (e *Element) Next() *Element {
	return (*Element)(atomic.LoadPointer(&e.next))
}
func (e *Element) addNext(next *Element) bool {
	return atomic.CompareAndSwapPointer(&e.next, nil, unsafe.Pointer(next))
}
func (e *Element) removeFromNext() (next *Element, ok bool) {
	for e.getFlag() == FlagOk {
		next = e.Next()
		if ok = atomic.CompareAndSwapPointer(&e.next, unsafe.Pointer(next), nil); ok {
			return next, ok
		}
	}
	return nil, false
}

type LinkedQueue struct {
	size       int32
	head, tail unsafe.Pointer
}

func (q *LinkedQueue) loadTail() *Element {
	return (*Element)(atomic.LoadPointer(&q.tail))
}
func (q *LinkedQueue) loadHead() *Element {
	return (*Element)(atomic.LoadPointer(&q.head))
}

func (q *LinkedQueue) Push(val interface{}) bool {
	newElem := newElement(val)

	for {
		currTail := q.loadTail()
		if currTail == nil { // 无元素
			if ok := atomic.CompareAndSwapPointer(&q.tail, nil, unsafe.Pointer(newElem)); ok {
				atomic.StorePointer(&q.head, unsafe.Pointer(newElem))
				return true
			}
		}
		if addOk := currTail.addNext(newElem); addOk {
			// Note: tail may point to a node which is not a tail node;
			// For when the new element is successfully appended, the next node may be pushed before the tail is set.
			atomic.StorePointer(&q.tail, unsafe.Pointer(newElem))
			atomic.AddInt32(&q.size, 1)
			return true
		}
		// otherwise continue to get the current tail
	}
}

func (q *LinkedQueue) Peek() (interface{}, bool) {
	// TODO
	return nil, false
}

func (q *LinkedQueue) Len() int {
	return int(atomic.LoadInt32(&q.size))
}
func (q *LinkedQueue) Pop() (interface{}, bool) {
	for {
		currHead := q.loadHead()
		if currHead == nil {
			return nil, false
		}

		if currHead.getFlag() == FlagDeleted {
			continue
		}
		next, removeOk := currHead.removeFromNext()
		if removeOk {
			currHead.setFlag(FlagDeleted)
			atomic.CompareAndSwapPointer(&q.head, unsafe.Pointer(currHead), unsafe.Pointer(next))
			atomic.AddInt32(&q.size, -1)

			return currHead.val, true
		} else {
			continue
		}
	}
}

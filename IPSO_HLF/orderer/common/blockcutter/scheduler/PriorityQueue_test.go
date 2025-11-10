package scheduler_test

import (
	"container/heap"
	"fmt"
	"testing"

	"github.com/Yunpeng-J/HLF-2.2/orderer/common/blockcutter/scheduler"
)

func Test1(t *testing.T) {
	var pq scheduler.PriorityQueue
	heap.Init(&pq)
	heap.Push(&pq, scheduler.NewTxNode(0, "123", nil))
	heap.Push(&pq, scheduler.NewTxNode(2, "456", nil))
	heap.Push(&pq, scheduler.NewTxNode(1, "007", nil))
	for pq.Len() > 0 {
		item := heap.Pop(&pq).(*scheduler.TxNode)
		fmt.Println(item)
	}

}

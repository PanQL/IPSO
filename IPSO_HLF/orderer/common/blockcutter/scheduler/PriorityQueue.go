package scheduler

import (
	"github.com/PanQL/fabric-protos-go/peer"
)

type TxNode struct {
	seq    int
	txid   string
	action *peer.ChaincodeAction
	index  int
}

func NewTxNode(seq int, txid string, action *peer.ChaincodeAction) *TxNode {
	return &TxNode{
		seq:    seq,
		txid:   txid,
		action: action,
		index:  -1,
	}
}

type PriorityQueue []*TxNode

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].seq < pq[j].seq
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*TxNode)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

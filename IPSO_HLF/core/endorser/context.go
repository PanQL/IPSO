package endorser

import (
	"container/heap"
	"log"
	"sync"
	"time"

	pb "github.com/PanQL/fabric-protos-go/peer"
)

type TxContext struct {
	proposal *UnpackedProposal
	response *pb.ProposalResponse
	err      error
	seq      int
	session  string
	ch       chan struct{}
}

type ContextManager struct {
	mutex    sync.Mutex
	contexts *PriorityQueue
	session  string
	nextSeq  int
	ch       chan *TxContext
}

func NewContextManager() *ContextManager {
	return &ContextManager{
		mutex:    sync.Mutex{},
		contexts: &PriorityQueue{},
		nextSeq:  0,
		ch:       make(chan *TxContext, 100000),
	}
}

func contextId(channelID, txID string) string {
	return channelID + txID
}

func (c *ContextManager) SetNext(seq int, session string) {
	log.Println("debug v17 set context", seq, session)
	c.mutex.Lock()
	c.nextSeq = seq
	c.session = session
	c.mutex.Unlock()
}

func (c *ContextManager) Create(seq int, session string, up *UnpackedProposal) *TxContext {
	// log.Printf("debug v18 context create %d", seq)
	var ctx *TxContext
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if seq == c.nextSeq {
		ctx = &TxContext{
			seq:      seq,
			proposal: up,
			response: nil,
			err:      nil,
			session:  session,
			ch:       make(chan struct{}),
		}
		c.ch <- ctx
		c.nextSeq += 1
		for c.contexts.Len() > 0 && (*c.contexts)[0].seq == c.nextSeq {
			c.ch <- heap.Pop(c.contexts).(*TxContext)
			c.nextSeq += 1
		}
		// log.Printf("debug v17 update context 1 next %d", c.nextSeq)

	} else if seq > c.nextSeq || seq < 100 { // adhoc
		ctx = &TxContext{
			seq:      seq,
			proposal: up,
			response: nil,
			err:      nil,
			session:  session,
			ch:       make(chan struct{}),
		}
		heap.Push(c.contexts, ctx)
		if c.contexts.Len() > 0 && (*c.contexts)[0].seq == c.nextSeq {
			for c.contexts.Len() > 0 && (*c.contexts)[0].seq == c.nextSeq {
				c.ch <- heap.Pop(c.contexts).(*TxContext)
				c.nextSeq += 1
			}
			// 		} else {
			// 			log.Printf("debug v17 next %d head %d receive %d", c.nextSeq, (*c.contexts)[0].seq, seq)
			//
		}
	} else {
		log.Printf("debug v17 drop stale txn %d cur %d", seq, c.nextSeq)
	}
	return ctx
}

func (c *ContextManager) Get() *TxContext {
	var res *TxContext
	for {
		select {
		case res = <-c.ch:
			// log.Printf("debug v17 get context %d %s", res.seq, res.session)
			return res
		case <-time.After(time.Duration(1) * time.Second):
			c.mutex.Lock()
			if c.contexts.Len() > 0 {
				res = heap.Pop(c.contexts).(*TxContext)
				log.Printf("debug v19 update context from %d to %d", c.nextSeq, res.seq+1)
				c.nextSeq = res.seq + 1
			}
			c.mutex.Unlock()
			if res != nil {
				return res
			}
		}
	}
}

func (c *ContextManager) Delete(seq int) {

}

func (c *ContextManager) Close() {
	// TODO
}

type PriorityQueue []*TxContext

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].seq < pq[j].seq
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*TxContext)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*pq = old[0 : n-1]
	return item
}

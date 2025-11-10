package scheduler

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PanQL/fabric-protos-go/ledger/rwset"
	"github.com/PanQL/fabric-protos-go/peer"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/gogo/protobuf/proto"
)

var maxUniqueKeys = 2048 // 65563

type Scheduler struct {
	windowSize int

	sessionTxs       map[string][]*TxNode
	sessionNextSeq   map[string]int
	sessionFutureTxs map[string]*PriorityQueue
	uniqueKeyMap     map[string]int
	uniqueKeyCounter int

	Process_blk_latency int

	// totalTxns []string // Txids
}

func NewScheduler() *Scheduler {
	maxUniqueKeys, _ := strconv.Atoi(os.Getenv("maxUniqueKeys"))
	log.Println("debug v3, maxUniqueKeys", maxUniqueKeys)
	if maxUniqueKeys == 0 {
		log.Println("debug v3")
		maxUniqueKeys = 1024
	}
	windowSize, _ := strconv.Atoi(os.Getenv("windowSize"))
	if windowSize == 0 {
		log.Println("debug v3")
		windowSize = 1024 * 12
	}
	return &Scheduler{
		windowSize:       windowSize,
		sessionTxs:       map[string][]*TxNode{},
		sessionNextSeq:   map[string]int{},
		sessionFutureTxs: map[string]*PriorityQueue{},
	}
}

// func (scheduler *Scheduler) Pending() int {
// 	return len(scheduler.totalTxns)
// }

func (scheduler *Scheduler) Pending() int {
	res := 0
	for _, session := range scheduler.sessionTxs {
		res += len(session)
	}
	return res
}

func (scheduler *Scheduler) parseExecOrdFromAct(action *peer.ChaincodeAction) int32 {
	protoMsg := &rwset.TxReadWriteDeltaSet{}
	var err error
	if err = proto.Unmarshal(action.Results, protoMsg); err != nil {
		panic("Fail to parse chaincode action")
	}

	return protoMsg.ExecuteOrder
}

func (scheduler *Scheduler) parseAction(action *peer.ChaincodeAction) ([]uint64, []uint64, []uint64, []string, bool) {
	readSet := make([]uint64, maxUniqueKeys/64)
	writeSet := make([]uint64, maxUniqueKeys/64)
	deltaSet := make([]uint64, maxUniqueKeys/64)
	var depSet []string
	txRWDset := &rwsetutil.TxRwdSet{}
	if err := txRWDset.FromProtoBytes(action.Results); err != nil {
		panic("Fail to retrieve rwset from txn payload")
	}
	// log.Println("debug v1 length rwd sets ", len(txRWDset.NsRwdSets))
	if len(txRWDset.NsRwdSets) > 1 {
		// log.Println("debug v1 namespace of idx=0", txRWDset.NsRwdSets[0].NameSpace)
		ns := txRWDset.NsRwdSets[1]
		// log.Println("debug v1 namespace of idx=1", ns.NameSpace)
		if ns.NameSpace == "lscc" {
			//
			return nil, nil, nil, nil, true
		}
		for _, read := range ns.KvRwdSet.Reads {
			readkey := read.GetKey()
			// readver := read.GetVersion()
			valbytes := read.GetValue()
			var vval ledger.VersionedValue
			err := json.Unmarshal(valbytes, &vval)
			if err != nil {
				// e.g., when read a non-exist account, this will fail.
				// But it's an expected behavior.
				continue
				// panic(fmt.Sprintf("unmarshal error when extract version from value in readset: %v", err))
			}
			depSet = append(depSet, vval.Txid)
			key, ok := scheduler.uniqueKeyMap[readkey]
			if ok == false {
				scheduler.uniqueKeyMap[readkey] = scheduler.uniqueKeyCounter
				key = scheduler.uniqueKeyCounter
				scheduler.uniqueKeyCounter += 1
			}
			if key >= maxUniqueKeys {
				// cut the block, and re-process this transaction in the next block
				log.Println("debug v1 drop transaction because read keys are overflow")
				return nil, nil, nil, nil, false
			}
			index := key / 64
			readSet[index] |= uint64(1) << (key % 64)
		}
		for _, write := range ns.KvRwdSet.Writes {
			writekey := write.GetKey()
			key, ok := scheduler.uniqueKeyMap[writekey]
			if ok == false {
				scheduler.uniqueKeyMap[writekey] = scheduler.uniqueKeyCounter
				key = scheduler.uniqueKeyCounter
				scheduler.uniqueKeyCounter += 1
			}
			if key >= maxUniqueKeys {
				// cut the block, and re-process this transaction in the next
				// block
				log.Println("debug v1 drop transaction because write keys are overflow")
				return nil, nil, nil, nil, false
			}
			index := key / 64
			writeSet[index] |= uint64(1) << (key % 64)
		}
		for _, delta := range ns.KvRwdSet.Deltas {
			deltakey := delta.GetKey()
			key, ok := scheduler.uniqueKeyMap[deltakey]
			if ok == false {
				scheduler.uniqueKeyMap[deltakey] = scheduler.uniqueKeyCounter
				key = scheduler.uniqueKeyCounter
				scheduler.uniqueKeyCounter += 1
			}
			if key >= maxUniqueKeys {
				// cut the block, and re-process this transaction in the next block
				log.Println("debug v1 drop transaction because delta keys are overflow")
				return nil, nil, nil, nil, false
			}
			index := key / 64
			deltaSet[index] |= uint64(1) << (key % 64)
		}
	}
	return readSet, writeSet, deltaSet, depSet, true
}

// func (scheduler *Scheduler) Schedule1(action *peer.ChaincodeAction, txId string) bool {
// 	scheduler.totalTxns = append(scheduler.totalTxns, txId)

// 	return true
// }

func (scheduler *Scheduler) Schedule(action *peer.ChaincodeAction, txId string) bool {
	// log.Println("debug v1 unique key length", scheduler.uniqueKeyCounter)
	temp := strings.Split(txId, "_+=+_")
	seq := -1
	session := "unknown"
	var err error
	if len(temp) == 3 {
		// seq, err = strconv.Atoi(temp[0])
		seq = int(scheduler.parseExecOrdFromAct(action))
		if err != nil {
			panic(fmt.Sprintf("failed to extract sequence number from transaction %s", txId))
		}
		session = temp[1]
	}
	log.Printf("Schedule seq=%d, session=%s, txid=%s", seq, session, txId)
	if session == "unknown" {
		scheduler.sessionTxs[session] = append(scheduler.sessionTxs[session], NewTxNode(seq, txId, action))
		return true
	} else {
		_, ok := scheduler.sessionNextSeq[session]
		if ok == false {
			scheduler.sessionNextSeq[session] = 0
		}
		if scheduler.sessionNextSeq[session] == seq {
			scheduler.sessionTxs[session] = append(scheduler.sessionTxs[session], NewTxNode(seq, txId, action))
			last := seq
			curFuture, ok := scheduler.sessionFutureTxs[session]
			if ok {
				for (curFuture.Len() > 0) && ((*curFuture)[0].seq == last+1) {
					scheduler.sessionTxs[session] = append(scheduler.sessionTxs[session], heap.Pop(curFuture).(*TxNode))
					last += 1
				}
			}
			scheduler.sessionNextSeq[session] = last + 1
		} else if scheduler.sessionNextSeq[session] < seq {
			// future transactions
			if pq, ok := scheduler.sessionFutureTxs[session]; ok {
				// window size
				if pq.Len() == 0 || (*pq)[0].seq+scheduler.windowSize >= seq {
					// log.Println("debug v1 append to future session", session, seq)
					heap.Push(pq, NewTxNode(seq, txId, action))
				} else {
					// drop this transaction
					// log.Println("debug v1 drop transaction, seq is too high", txId)
					return false
				}
			} else {
				// log.Println("debug v1 create heap, insert transaction", txId)
				pq := PriorityQueue{}
				heap.Push(&pq, NewTxNode(seq, txId, action))
				scheduler.sessionFutureTxs[session] = &pq
			}
		} else {
			session = "stale"
			_, ok := scheduler.sessionTxs[session]
			if !ok {
				scheduler.sessionTxs[session] = []*TxNode{}
			}
			scheduler.sessionTxs[session] = append(scheduler.sessionTxs[session], NewTxNode(seq, txId, action))
			log.Printf("debug v17 drop stale transaction %d in orderer", seq)
		}
	}

	return true
}

func (scheduler *Scheduler) ScheduleOld(action *peer.ChaincodeAction, txId string) bool {
	// log.Println("debug v1 unique key length", scheduler.uniqueKeyCounter)
	temp := strings.Split(txId, "_+=+_")
	seq := -1
	session := "unknown"
	var err error
	if len(temp) == 3 {
		seq, err = strconv.Atoi(temp[0])
		if err != nil {
			panic(fmt.Sprintf("failed to extract sequence number from transaction %s", txId))
		}
		session = temp[1]
	}
	if session == "unknown" {
		scheduler.sessionTxs[session] = append(scheduler.sessionTxs[session], NewTxNode(seq, txId, action))
		return true
	} else {
		_, ok := scheduler.sessionNextSeq[session]
		if ok == false {
			scheduler.sessionNextSeq[session] = 0
		}
		if scheduler.sessionNextSeq[session] == seq {
			scheduler.sessionTxs[session] = append(scheduler.sessionTxs[session], NewTxNode(seq, txId, action))
			last := seq
			curFuture, ok := scheduler.sessionFutureTxs[session]
			if ok {
				for (curFuture.Len() > 0) && ((*curFuture)[0].seq == last+1) {
					scheduler.sessionTxs[session] = append(scheduler.sessionTxs[session], heap.Pop(curFuture).(*TxNode))
					last += 1
				}
			}
			scheduler.sessionNextSeq[session] = last + 1
		} else if scheduler.sessionNextSeq[session] < seq {
			// future transactions
			if pq, ok := scheduler.sessionFutureTxs[session]; ok {
				// window size
				if pq.Len() == 0 || (*pq)[0].seq+scheduler.windowSize >= seq {
					// log.Println("debug v1 append to future session", session, seq)
					heap.Push(pq, NewTxNode(seq, txId, action))
				} else {
					// drop this transaction
					// log.Println("debug v1 drop transaction, seq is too high", txId)
					return false
				}
			} else {
				// log.Println("debug v1 create heap, insert transaction", txId)
				pq := PriorityQueue{}
				heap.Push(&pq, NewTxNode(seq, txId, action))
				scheduler.sessionFutureTxs[session] = &pq
			}
		} else {
			session = "stale"
			_, ok := scheduler.sessionTxs[session]
			if !ok {
				scheduler.sessionTxs[session] = []*TxNode{}
			}
			scheduler.sessionTxs[session] = append(scheduler.sessionTxs[session], NewTxNode(seq, txId, action))
			log.Printf("debug v17 drop stale transaction %d in orderer", seq)
		}
	}

	return true
}

// func (scheduler *Scheduler) ProcessBlk1() (result []string, invalidTxns []string) {
// 	result = scheduler.totalTxns
// 	scheduler.totalTxns = []string{}

// 	return result, invalidTxns
// }

func (scheduler *Scheduler) ProcessBlk() (result []string, invalidTxns []string) {
	defer func() {
		scheduler.sessionTxs = map[string][]*TxNode{}
	}()

	sessionNames := []string{}
	for k, _ := range scheduler.sessionTxs {
		sessionNames = append(sessionNames, k)
	}
	for k, _ := range scheduler.sessionFutureTxs {
		sessionNames = append(sessionNames, k)
	}
	// sort sessionNames to guarantee determinism
	sort.Strings(sessionNames)

	last := ""
	for _, sessionName := range sessionNames {
		if sessionName == last {
			continue
		}
		last = sessionName

		for _, v := range scheduler.sessionTxs[sessionName] {
			result = append(result, v.txid)
		}
		if pq, ok := scheduler.sessionFutureTxs[sessionName]; ok {
			next := scheduler.sessionNextSeq[sessionName]
			if pq.Len() > 0 && next < (*pq)[0].seq { // adhoc
				next = (*pq)[0].seq
			}
			for pq.Len() > 0 && next == (*pq)[0].seq {
				result = append(result, heap.Pop(pq).(*TxNode).txid)
				next += 1
			}
			scheduler.sessionNextSeq[sessionName] = next
			if pq.Len() == 0 {
				delete(scheduler.sessionFutureTxs, sessionName)
			}
		}
	}

	return result, invalidTxns
}

func (scheduler *Scheduler) ProcessBlkOld() (result []string, invalidTxns []string) {
	now := time.Now()
	defer func() {
		sec := time.Since(now).Milliseconds()
		scheduler.Process_blk_latency = int(sec)
		log.Printf("benchmark ProcessBlk in %d ms\n", sec)
		// clear scheduler
		scheduler.sessionTxs = map[string][]*TxNode{}
		scheduler.uniqueKeyCounter = 0
		scheduler.uniqueKeyMap = map[string]int{}

	}()

	sessionNames := []string{}
	for k, _ := range scheduler.sessionTxs {
		sessionNames = append(sessionNames, k)
	}
	for k, _ := range scheduler.sessionFutureTxs {
		sessionNames = append(sessionNames, k)
	}
	// sort sessionNames to guarantee determinism
	sort.Strings(sessionNames)
	// log.Println("debug v1 session names", sessionNames)
	// log.Println("debug v3", time.Since(now).Milliseconds())

	__temp := time.Now()

	// build node (one node may contain multiple transactions)
	var allNodes []*Node
	numOfNodes := int32(0)
	last := ""
	for _, sessionName := range sessionNames {
		if sessionName == last {
			continue
		}
		last = sessionName

		// for each session
		var txs []*TxNode
		var nodes []*Node
		for _, v := range scheduler.sessionTxs[sessionName] {
			txs = append(txs, v)
		}
		if pq, ok := scheduler.sessionFutureTxs[sessionName]; ok {
			next := scheduler.sessionNextSeq[sessionName]
			if pq.Len() > 64 && next < (*pq)[0].seq { // adhoc
				log.Printf("debug v17 drop transactions from %d to %d", next, (*pq)[0].seq)
				next = (*pq)[0].seq
			}
			for pq.Len() > 0 && next == (*pq)[0].seq {
				txs = append(txs, heap.Pop(pq).(*TxNode))
				next += 1
			}
			scheduler.sessionNextSeq[sessionName] = next
			if pq.Len() == 0 {
				delete(scheduler.sessionFutureTxs, sessionName)
			}
		}
		// log.Printf("debug v2 number of transactions %d in session %s\n", len(txs), sessionName)
		// reverse
		// for i, j := 0, len(txs)-1; i < j; i, j = i+1, j-1 {
		//	txs[i], txs[j] = txs[j], txs[i]
		// }
		for i := 0; i < len(txs); i += 1 {
			//for i := len(txs) - 1; i >= 0; i -= 1 {
			tx := txs[i]
			rs, ws, ds, dep, ok := scheduler.parseAction(tx.action)
			if ok == false {
				// log.Println("debug v1 parse ation return false for transaction", tx.txid)
				continue
			}
			// log.Printf("debug v2 tx %s dependency %v\n", tx.txid, dep)
			var found []int
			for j := 0; j < len(nodes); j++ {
				if nodes[j] == nil {
					// be nil because of the following merge
					continue
				}
				cur := nodes[j]
				// TODO: optimize
				for _, dp := range dep {
					for _, item := range cur.txids {
						if dp == item {
							found = append(found, j)
						}
					}
				}
			}
			mergeNodes := func() {
				// log.Println("debug v2 merge nodes", i, found)
				node := Node{
					index:    -1,
					txids:    []string{tx.txid},
					readSet:  rs,
					writeSet: ws,
					deltaSet: ds,
				}
				for _, idx := range found {
					cur := nodes[idx]
					// merge
					if cur == nil {
						continue // found is a muptiset
						// panic("debug v7 node is nil")
					}
					node.txids = append(node.txids, cur.txids...)
					for k := 0; k < (maxUniqueKeys / 64); k++ {
						node.readSet[k] |= cur.readSet[k]
						node.writeSet[k] |= cur.writeSet[k]
						node.deltaSet[k] |= cur.deltaSet[k]
					}
					nodes[idx] = nil
				}
				// log.Println("debug v2 merge txids:", node.txids)
				nodes = append(nodes, &node)
			}

			if len(found) == 0 {
				node := &Node{
					index:    -1,
					txids:    []string{tx.txid},
					readSet:  rs,
					writeSet: ws,
					deltaSet: ds,
				}
				nodes = append(nodes, node)
			} else {
				// merge nodes
				mergeNodes()
			}
		}
		for _, node := range nodes {
			if node != nil {
				node.index = numOfNodes
				node.weight = len(node.txids)
				allNodes = append(allNodes, node)
				numOfNodes += 1
			}
		}
	}
	log.Printf("benchmark build node in %d ms", time.Since(__temp).Milliseconds())
	__temp = time.Now()

	log.Println("benchmark number of nodes", numOfNodes)
	log.Println("benchmark number of keys", scheduler.uniqueKeyCounter)

	// build dependency graph
	graph := make([][]int32, numOfNodes)
	invgraph := make([][]int32, numOfNodes)
	for i := int32(0); i < numOfNodes; i++ {
		graph[i] = make([]int32, 0, numOfNodes)
		invgraph[i] = make([]int32, 0, numOfNodes)
	}
	// log.Printf("benchmark allocate memory in %d ms", time.Since(__temp).Milliseconds())
	for i := int32(0); i < numOfNodes; i++ {
		for j := int32(0); j < numOfNodes; j++ {
			if i == j {
				continue
			}
			for k := 0; k < (scheduler.uniqueKeyCounter / 64); k++ {
				if allNodes[i].writeSet[k]&allNodes[j].readSet[k] != 0 {
					graph[i] = append(graph[i], j)
					invgraph[j] = append(invgraph[j], i)
					break
				}
				if allNodes[i].deltaSet[k]&allNodes[j].readSet[k] != 0 {
					graph[i] = append(graph[i], j)
					invgraph[j] = append(invgraph[j], i)
					break
				}
			}
		}
	}
	log.Printf("benchmark build graph in %d ms", time.Since(__temp).Milliseconds())
	__temp = time.Now()
	// log.Printf("debug v2 graph\n")
	// for i := int32(0); i < numOfNodes; i++ {
	// 	ps := fmt.Sprintf("debug v2 Node weight %d nodeid %d ", len(allNodes[i].txids), i)
	// 	for j := 0; j < len(graph[i]); j++ {
	// 		ps = ps + fmt.Sprintf("%d ", graph[i][j])
	// 	}
	// 	log.Println(ps)
	// }

	// schedule
	schedule, invSet := scheduler.getSchedule(&graph, &invgraph, &allNodes)
	nodeLen := len(schedule)
	// log.Println("debug v3 length of schedule", nodeLen)
	// log.Println("debug v3 invSet", invSet)

	for i := 0; i < nodeLen; i += 1 {
		txids := allNodes[schedule[nodeLen-1-i]].txids
		for j, _ := range txids {
			result = append(result, txids[len(txids)-1-j])
		}
	}
	for i := int32(0); i < numOfNodes; i += 1 {
		if invSet[i] == true {
			txids := allNodes[i].txids
			for j, _ := range txids {
				invalidTxns = append(invalidTxns, txids[len(txids)-1-j])
			}
		}
	}
	log.Printf("benchmark schedule in %d ms", time.Since(__temp).Milliseconds())

	return result, invalidTxns
}

func (scheduler *Scheduler) getSchedule(graph *[][]int32, invgraph *[][]int32, allNodes *[]*Node) ([]int32, []bool) {
	// dagGenerator := NewJohnsonCE(graph, allNodes)
	dagGenerator := NewFVS(graph, allNodes)
	invCount, invSet := dagGenerator.Run()
	// log.Printf("debug v2 invalid ndoes: %v", invSet)
	invtemp := "debug v2 Node invalid"
	for i, ok := range invSet {
		if ok {
			invtemp += fmt.Sprintf(" %d", i)
		}
	}
	log.Println(invtemp)
	n := int32(len(*graph))
	visited := make([]bool, n)
	schedule := make([]int32, 0, n-invCount)
	remainingNode := n - invCount
	cur := int32(0)
	for remainingNode != 0 {
		// log.Printf("debug v2: remainingNode %d, cur %d", remainingNode, cur)
		flag := true
		if visited[cur] || invSet[cur] {
			cur = (cur + 1) % n
			continue
		}
		for _, in := range (*invgraph)[cur] {
			if (visited[in] || invSet[in]) == false {
				cur = in
				flag = false
				break
			}
		}
		if flag {
			visited[cur] = true
			remainingNode -= 1
			schedule = append(schedule, cur)
			for _, next := range (*graph)[cur] {
				if (visited[next] || invSet[next]) == false {
					cur = next
					break
				}
			}
		}
	}
	return schedule, invSet
}

func (scheduler *Scheduler) checkDependency(i int32) bool {
	// TODO: check if scheduler.pendingTxns[i]'s dependent transactions exist
	// for _, txid := range scheduler.txReadDep[i] {
	// 	if idx, ok := scheduler.txidMap[txid]; ok {
	// 		// its dependent transaction exists in the current block
	// 		if scheduler.invalidTxs[idx] == true {
	// 			return false
	// 		}
	// 	} else {
	// 		temp := strings.Split(txid, "_+=+_")
	// 		seq := -1
	// 		session := ""
	// 		var err error
	// 		if len(temp) == 3 {
	// 			seq, err = strconv.Atoi(temp[0])
	// 			if err != nil {
	// 				panic(fmt.Sprintf("failed to extract sequence from transaction %s", txid))
	// 			}
	// 			session = temp[1]
	// 		}
	// 		for _, wait := range scheduler.sessionWait[session] {
	// 			if wait == int32(seq) {
	// 				return false
	// 			}
	// 		}
	// 	}
	// }
	return true
}

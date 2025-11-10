package infra

import (
	"fmt"
	"sort"
	"time"

	"github.com/PanQL/fabric-protos-go/peer"
)

var (
	timeKeepers TimeKeepers
)

type TimeKeepers struct {
	transactions        []*TimeKeeper
	endorseLatency      []int64
	integrateLatency    []int64
	orderCommitLatency  []int64
	totalLatency        []int64
	commitLatencySorted []int64
	retryTimesSorted    []int
	waited              int32
	historicalTKs       [][]*TimeKeeper // historical timekeepers for each transaction
}

type TimeKeeper struct {
	Txid          string
	ProposedTime  int64
	EndorsedTime  int64
	BroadcastTime int64
	ObservedTime  int64
}

func initTimeKeepers(Txids *[]string) {
	timeKeepers = TimeKeepers{
		transactions:        make([]*TimeKeeper, config.TxNum),
		endorseLatency:      make([]int64, config.TxNum),
		integrateLatency:    make([]int64, config.TxNum),
		orderCommitLatency:  make([]int64, config.TxNum),
		totalLatency:        make([]int64, config.TxNum),
		retryTimesSorted:    nil,
		commitLatencySorted: nil,
		waited:              0,
		historicalTKs:       make([][]*TimeKeeper, config.TxNum),
	}
	for i := range timeKeepers.transactions {
		timeKeepers.transactions[i] = &TimeKeeper{Txid: (*Txids)[i], ProposedTime: 0, EndorsedTime: 0, BroadcastTime: 0, ObservedTime: 0}
	}
}

// return (abort rate, average endorsement lat, average integration lat, average order&commit lat, average total lat)
func (tks *TimeKeepers) calculateRound0Stats() (abortRate float64, averageEndorseLatency int64, averageIntegrateLatency int64, averageOrderCommitLatency int64, averageTotalLatency int64, successTxns []string) {
	validNum := 0
	endorseTime := int64(0)
	integrateTime := int64(0)
	orderCommitTime := int64(0)
	totalTime := int64(0)

	for i, tk := range tks.transactions {
		if len(tks.historicalTKs[i]) > 0 {
			continue
		}
		successTxns = append(successTxns, tk.Txid)
		validNum += 1
		endorseTime += tk.EndorsedTime - tk.ProposedTime
		integrateTime += tk.BroadcastTime - tk.EndorsedTime
		orderCommitTime += tk.ObservedTime - tk.BroadcastTime
		totalTime += tk.ObservedTime - tk.ProposedTime
	}
	abortRate = float64(config.TxNum-validNum) / float64(config.TxNum)
	averageEndorseLatency = endorseTime / int64(validNum)
	averageIntegrateLatency = integrateTime / int64(validNum)
	averageOrderCommitLatency = orderCommitTime / int64(validNum)
	averageTotalLatency = totalTime / int64(validNum)

	return
}

func (tks *TimeKeepers) calculateObservedTimes() {
	for i, cur_tk := range tks.transactions {
		cur_tk.ObservedTime = observedTimes[cur_tk.Txid]
		timeKeepers.totalLatency[i] += cur_tk.ObservedTime - cur_tk.ProposedTime
		timeKeepers.orderCommitLatency[i] += cur_tk.ObservedTime - cur_tk.BroadcastTime
		for _, historical_tk := range tks.historicalTKs[i] {
			historical_tk.ObservedTime = observedTimes[historical_tk.Txid]
			timeKeepers.totalLatency[i] += historical_tk.ObservedTime - historical_tk.ProposedTime
			timeKeepers.orderCommitLatency[i] += historical_tk.ObservedTime - historical_tk.BroadcastTime
		}
	}
}

func (tks *TimeKeepers) keepWaitedTime(time int32) {
	tks.waited = time
}

func (tks *TimeKeepers) keepProposedTime(
	id int,
	txid string,
	endorserIndex int,
	connIndex int,
	clientIndex int,
) {
	proposedTime := time.Now().UnixNano()

	// id := txid2id[txid]
	logCh <- fmt.Sprintf("%-10s %d %4d %s %d %d %d", "Proposed", proposedTime, id, txid, endorserIndex, connIndex, clientIndex)

	if timeKeepers.transactions[id].ProposedTime != 0 {
		tks.historicalTKs[id] = append(tks.historicalTKs[id], tks.transactions[id])
		tks.transactions[id] = &TimeKeeper{Txid: txid, ProposedTime: 0, EndorsedTime: 0, BroadcastTime: 0, ObservedTime: 0}
	}
	timeKeepers.transactions[id].ProposedTime = proposedTime
}

func (tks *TimeKeepers) keepEndorsedTime(
	id int,
	txid string,
	endorserIndex int,
	connIndex int,
	clientIndex int,
) {
	endorsedTime := time.Now().UnixNano()

	// id := txid2id[txid]
	logCh <- fmt.Sprintf("%-10s %d %4d %s %d %d %d", "Endorsed", endorsedTime, id, txid, endorserIndex, connIndex, clientIndex)

	timeKeepers.transactions[id].EndorsedTime = endorsedTime
	timeKeepers.endorseLatency[id] += endorsedTime - timeKeepers.transactions[id].ProposedTime
}

func (tks *TimeKeepers) keepBroadcastTime(
	id int,
	txid string,
	broadcasterIndex int,
) {
	broadcastTime := time.Now().UnixNano()

	// id := txid2id[txid]
	logCh <- fmt.Sprintf("%-10s %d %4d %s %d", "Broadcast", broadcastTime, id, txid, broadcasterIndex)

	timeKeepers.transactions[id].BroadcastTime = broadcastTime
	timeKeepers.integrateLatency[id] += broadcastTime - timeKeepers.transactions[id].EndorsedTime
}

func (tks *TimeKeepers) keepObservedTime(
	txid string,
	validationCode peer.TxValidationCode,
) {
	observedTime := time.Now().UnixNano()

	// id := txid2id[txid]
	logCh <- fmt.Sprintf("%-10s %d %s %s", "Observed", observedTime, txid, validationCode)
	observedTimes[txid] = observedTime

	// timeKeepers.transactions[id].ObservedTime = observedTime
	// timeKeepers.totalLatency[id] += observedTime - timeKeepers.transactions[id].ProposedTime
	// timeKeepers.orderCommitLatency[id] += observedTime - timeKeepers.transactions[id].BroadcastTime
}

func (tks *TimeKeepers) getAverageTotalLatency() float64 {
	// var result int64 = 0
	// for _, cl := range tks.totalLatency {
	// 	result += cl
	// }
	// return float64(result) / float64(config.TxNum) / 1e9
	// return tks.getAverageLatencyFromSlice(&tks.totalLatency)
	count := 0
	accumulate := int64(0)
	for _, lat := range tks.totalLatency {
		if lat == 0 {
			continue
		}
		count++
		accumulate += lat
	}
	if count == 0 {
		return 0
	}
	return float64(accumulate) / float64(count) / 1e9
}

func (tks *TimeKeepers) getAverageEndorseLatency() float64 {
	// var result int64 = 0
	// for _, cl := range tks.endorseLatency {
	// 	result += cl
	// }
	// return float64(result) / float64(config.TxNum) / 1e9
	return tks.getAverageLatencyFromSlice(&tks.endorseLatency)
}

func (tks *TimeKeepers) getAverageOrderCommitLatency() float64 {
	// var result int64 = 0
	// for _, cl := range tks.orderCommitLatency {
	// 	result += cl
	// }
	// return float64(result) / float64(config.TxNum) / 1e9
	return tks.getAverageLatencyFromSlice(&tks.orderCommitLatency)
}

func (tks *TimeKeepers) getAverageLatencyFromSlice(slice *[]int64) float64 {
	var result int64 = 0
	for _, cl := range *slice {
		result += cl
	}
	return float64(result) / float64(config.TxNum) / 1e9
}

func (tks *TimeKeepers) getCommitLatencyOfPercentile(p int) float64 {
	if tks.commitLatencySorted == nil {
		tks.sortCommitLatency()
	}

	index := int(float64(p) / 100.0 * float64(config.TxNum))
	if index < 0 {
		index = 0
	} else if index >= config.TxNum {
		index = config.TxNum - 1
	}

	return float64(tks.commitLatencySorted[index]) / 1e9
}

func (tks *TimeKeepers) sortCommitLatency() {
	tks.commitLatencySorted = make([]int64, len(tks.totalLatency))
	copy(tks.commitLatencySorted, tks.totalLatency)
	sort.Slice(
		tks.commitLatencySorted,
		func(i, j int) bool {
			return tks.commitLatencySorted[i] < tks.commitLatencySorted[j]
		},
	)
}

func (tks *TimeKeepers) getRetryTimesOfPercentile(p int) int {
	if tks.retryTimesSorted == nil {
		tks.sortRetryTimes()
	}

	index := int(float64(p) / 100.0 * float64(config.TxNum))
	if index < 0 {
		index = 0
	} else if index >= config.TxNum {
		index = config.TxNum - 1
	}

	return tks.retryTimesSorted[index]
}

func (tks *TimeKeepers) sortRetryTimes() {
	tks.retryTimesSorted = make([]int, len(tks.totalLatency))
	for i, _ := range tks.transactions {
		tks.retryTimesSorted[i] = len(tks.historicalTKs[i])
	}
	sort.Slice(
		tks.retryTimesSorted,
		func(i, j int) bool {
			return tks.retryTimesSorted[i] < tks.retryTimesSorted[j]
		},
	)
}

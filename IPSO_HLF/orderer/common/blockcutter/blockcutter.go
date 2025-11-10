/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockcutter

import (
	"log"
	"os"
	"strconv"
	"time"

	cb "github.com/PanQL/fabric-protos-go/common"
	"github.com/PanQL/fabric-protos-go/peer"
	"github.com/Yunpeng-J/HLF-2.2/common/channelconfig"
	"github.com/Yunpeng-J/HLF-2.2/common/flogging"
	"github.com/Yunpeng-J/HLF-2.2/orderer/common/blockcutter/scheduler"
	utils "github.com/Yunpeng-J/HLF-2.2/protoutil"
)

var logger = flogging.MustGetLogger("orderer.common.blockcutter")

type OrdererConfigFetcher interface {
	OrdererConfig() (channelconfig.Orderer, bool)
}

// Receiver defines a sink for the ordered broadcast messages
type Receiver interface {
	// Ordered should be invoked sequentially as messages are ordered
	// Each batch in `messageBatches` will be wrapped into a block.
	// `pending` indicates if there are still messages pending in the receiver.
	Ordered(msg *cb.Envelope) (messageBatches [][]*cb.Envelope, pending bool)

	// Cut returns the current batch and starts a new one
	Cut() []*cb.Envelope
}

type receiver struct {
	sharedConfigFetcher OrdererConfigFetcher
	// optimistic code begin
	// pendingBatch []*cb.Envelope
	pendingBatch           map[string]*cb.Envelope
	pendingBatchNonEndorse []*cb.Envelope
	scheduler              *scheduler.Scheduler
	// optimistic code end
	pendingBatchSizeBytes uint32

	PendingBatchStartTime time.Time
	ChannelID             string
	Metrics               *Metrics
}

// NewReceiverImpl creates a Receiver implementation based on the given configtxorderer manager
func NewReceiverImpl(channelID string, sharedConfigFetcher OrdererConfigFetcher, metrics *Metrics) Receiver {
	return &receiver{
		sharedConfigFetcher:    sharedConfigFetcher,
		Metrics:                metrics,
		ChannelID:              channelID,
		pendingBatchNonEndorse: make([]*cb.Envelope, 0),
		pendingBatch:           make(map[string]*cb.Envelope),
		pendingBatchSizeBytes:  0,
		scheduler:              scheduler.NewScheduler(),
	}
}

// optimistic code begin
// ScheduleMsg
func (r *receiver) ScheduleMsg(msg *cb.Envelope) bool {
	payload, err := utils.UnmarshalPayload(msg.GetPayload())
	if err != nil {
		panic("Can not get payload from the txn envelop: ")
	}
	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		panic("Can not mershal channel header from the txn payload")
	}
	if cb.HeaderType(chdr.Type) != cb.HeaderType_ENDORSER_TRANSACTION {
		r.pendingBatchNonEndorse = append(r.pendingBatchNonEndorse, msg)
		logger.Infof("Put ahead non-endorsement txn %s\n\n", chdr.TxId[0:8])
		return true
	} else {
		var respPayload *peer.ChaincodeAction
		if respPayload, err = utils.GetActionFromEnvelopeMsg(msg); err != nil {
			panic("Fail to get action from the txn envelop")
		}

		if r.scheduler.Schedule(respPayload, chdr.TxId) {
			r.pendingBatch[chdr.TxId] = msg
			return true
		} else {
			log.Printf("debug v7 drop transaction %s", chdr.TxId)
			return false
		}
	}
}

// optimistic code end

// Ordered should be invoked sequentially as messages are ordered
//
// messageBatches length: 0, pending: false
//   - impossible, as we have just received a message
// messageBatches length: 0, pending: true
//   - no batch is cut and there are messages pending
// messageBatches length: 1, pending: false
//   - the message count reaches BatchSize.MaxMessageCount
// messageBatches length: 1, pending: true
//   - the current message will cause the pending batch size in bytes to exceed BatchSize.PreferredMaxBytes.
// messageBatches length: 2, pending: false
//   - the current message size in bytes exceeds BatchSize.PreferredMaxBytes, therefore isolated in its own batch.
// messageBatches length: 2, pending: true
//   - impossible
//
// Note that messageBatches can not be greater than 2.
func (r *receiver) Ordered(msg *cb.Envelope) (messageBatches [][]*cb.Envelope, pending bool) {
	if len(r.pendingBatch) == 0 {
		// We are beginning a new batch, mark the time
		r.PendingBatchStartTime = time.Now()
	}

	ordererConfig, ok := r.sharedConfigFetcher.OrdererConfig()
	if !ok {
		logger.Panicf("Could not retrieve orderer config to query batch parameters, block cutting is not possible")
	}

	batchSize := ordererConfig.BatchSize()

	messageSizeBytes := messageSizeBytes(msg)
	if messageSizeBytes > batchSize.PreferredMaxBytes {
		logger.Debugf("The current message, with %v bytes, is larger than the preferred batch size of %v bytes and will be isolated.", messageSizeBytes, batchSize.PreferredMaxBytes)

		// cut pending batch, if it has any messages
		if len(r.pendingBatch) > 0 {
			messageBatch := r.Cut()
			messageBatches = append(messageBatches, messageBatch)
		}

		// create new batch with single message
		messageBatches = append(messageBatches, []*cb.Envelope{msg})

		// Record that this batch took no time to fill
		r.Metrics.BlockFillDuration.With("channel", r.ChannelID).Observe(0)

		return
	}

	messageWillOverflowBatchSizeBytes := r.pendingBatchSizeBytes+messageSizeBytes > batchSize.PreferredMaxBytes

	if messageWillOverflowBatchSizeBytes {
		logger.Debugf("The current message, with %v bytes, will overflow the pending batch of %v bytes.", messageSizeBytes, r.pendingBatchSizeBytes)
		logger.Debugf("Pending batch would overflow if current message is added, cutting batch now.")
		messageBatch := r.Cut()
		r.PendingBatchStartTime = time.Now()
		messageBatches = append(messageBatches, messageBatch)
	}

	logger.Debugf("Enqueuing message into batch")
	// optimistic code begin
	if !r.ScheduleMsg(msg) {
		pending = false
		return
	}
	r.pendingBatchSizeBytes += messageSizeBytes
	pending = true

	// if uint32(len(r.pendingBatch)) >= batchSize.MaxMessageCount {
	queueLen := r.scheduler.Pending()
	if uint32(queueLen) >= batchSize.MaxMessageCount {
		logger.Infof("Batch size met, cutting batch %d", queueLen)
		messageBatch := r.Cut()
		messageBatches = append(messageBatches, messageBatch)
		pending = false
	}

	return
}

// Cut returns the current batch and starts a new one
func (r *receiver) Cut() []*cb.Envelope {
	if len(r.pendingBatch) != 0 {
		r.Metrics.BlockFillDuration.With("channel", r.ChannelID).Observe(time.Since(r.PendingBatchStartTime).Seconds())
	}
	r.PendingBatchStartTime = time.Time{}
	// optimistic code begin
	// batch := r.pendingBatch
	batch := make([]*cb.Envelope, 0)
	batch = append(batch, r.pendingBatchNonEndorse...)
	r.pendingBatchNonEndorse = make([]*cb.Envelope, 0)
	st := time.Now()
	schedule, invalid := r.scheduler.ProcessBlk()
	r.Metrics.BlockScheduleDuration.With("channel", r.ChannelID).Observe(time.Since(st).Seconds())
	// log.Printf("debug v7 len_of_schedule %d len_of_invalid %d len_of_batching %d", len(schedule), len(invalid), len(r.pendingBatch))
	for _, txId := range schedule {
		if r.pendingBatch[txId] == nil {
			log.Printf("debug v7 pendingbatch valid nil %s", txId)
		}
		batch = append(batch, r.pendingBatch[txId])
		delete(r.pendingBatch, txId)
	}
	for _, txId := range invalid {
		if r.pendingBatch[txId] == nil {
			log.Printf("debug v7 pendingbatch invalid nil %s", txId)
		}
		batch = append(batch, r.pendingBatch[txId])
		delete(r.pendingBatch, txId)
	}
	// debug
	log.Printf("debug v1 length of invalid transactions %d, should be 0 in benign case\n", len(invalid))
	// for _, txid := range invalid {
	// 	log.Printf("debug v1 invalid transactions %s\n", txid)
	// }

	// r.pendingBatch = nil
	r.pendingBatchSizeBytes = 0 // TODO: should not be 0 (we do not use this condition to cut)
	// r.pendingBatch = make(map[string]*cb.Envelope)
	// optimistic code end
	slp, ok := os.LookupEnv("OrdererSleep")
	var ordererSleep int
	if ok {
		ordererSleep, _ = strconv.Atoi(slp)
		time.Sleep(time.Duration(ordererSleep-r.scheduler.Process_blk_latency) * time.Millisecond)
	}
	return batch
}

func messageSizeBytes(message *cb.Envelope) uint32 {
	return uint32(len(message.Payload) + len(message.Signature))
}

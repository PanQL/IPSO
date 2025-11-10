package infra

import (
	"strconv"

	"github.com/PanQL/fabric-protos-go/peer"
)

type Initiator struct {
	init_proposals []*peer.Proposal
	ccArgsList     [][]string
	txids          []string
	outCh          chan *Element
	inCh           chan string
	txid2id        map[string]int // Maps transaction ID to its index
	newIdGen       int            // the next id to generate new customal txid
}

func NewInitiator(inCh chan string, outCh chan *Element) *Initiator {
	it := &Initiator{
		init_proposals: make([]*peer.Proposal, config.TxNum),
		txids:          make([]string, config.TxNum),
		outCh:          outCh,
		inCh:           inCh,
		ccArgsList:     generateCCArgsList(),
		txid2id:        make(map[string]int, config.TxNum),
		newIdGen:       config.TxIDStart + config.TxNum,
	}

	// Create proposal and id for all generated transactions
	// ccArgsList := generateCCArgsList()
	session := getSession()
	for i := 0; i < config.TxNum; i++ {
		ccArgs := it.ccArgsList[i]

		tempTXID := ""
		if !config.CheckTxID {
			tempTXID = generateCustomTXID(i, session)
		}

		proposal, txID, err := CreateProposal(
			tempTXID,
			config.Channel,
			config.Chaincode,
			config.Version,
			ccArgs,
		)
		if err != nil {
			logger.Fatalf("Fail to create proposal %s: %v", txID, err)
		}

		it.txid2id[txID] = i
		it.init_proposals[i] = proposal
		it.txids[i] = txID
	}

	return it
}

func getSession() string {
	if config.Session != "" {
		return config.Session
	}
	return getName(20)
}

func generateCustomTXID(i int, session string) string {
	return strconv.Itoa(config.TxIDStart+i) + "_+=+_" + session + "_+=+_" + getName(20)
}

// StartSync sends all unsigned transactions (raw transactions) to the channel 'raw'
// waiting for subsequent processing
func (it *Initiator) StartSync() {
	for i := 0; i < len(it.init_proposals); i++ {
		it.outCh <- &Element{Proposal: it.init_proposals[i], Txid: it.txids[i], Id: i}
	}

	// it.End()
}

func (it *Initiator) StartAsync() {
	go it.Retry()
}

func (it *Initiator) Retry() {
	for {
		select {
		case abortedTxID := <-it.inCh:
			id := it.txid2id[abortedTxID]
			if id < 0 || id >= config.TxNum {
				logger.Fatalf("Invalid transaction id %d", id)
			}
			ccArgs := it.ccArgsList[id]
			session := getSession()

			tempTXID := ""
			if !config.CheckTxID {
				tempTXID = generateCustomTXID(it.newIdGen, session)
				it.newIdGen++
			}

			proposal, txID, err := CreateProposal(
				tempTXID,
				config.Channel,
				config.Chaincode,
				config.Version,
				ccArgs,
			)
			if err != nil {
				logger.Fatalf("Fail to create proposal %s: %v", txID, err)
			}
			it.txid2id[txID] = id
			it.outCh <- &Element{Proposal: proposal, Txid: txID, Id: id}
		case <-doneCh:
			return
		}
	}
}

// func (it *Initiator) End() {
// 	it.outCh <- nil
// }

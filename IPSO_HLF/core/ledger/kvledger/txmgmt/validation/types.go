//go:build !crdt
// +build !crdt

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation

import (
	"encoding/json"
	"log"

	"github.com/PanQL/fabric-protos-go/peer"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger/internal/version"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger/kvledger/txmgmt/rwsetutil"
)

// block is used to used to hold the information from its proto format to a structure
// that is more suitable/friendly for validation
type block struct {
	num uint64
	txs []*transaction
}

// transaction is used to hold the information from its proto format to a structure
// that is more suitable/friendly for validation
type transaction struct {
	indexInBlock            int
	id                      string
	rwdset                  *rwsetutil.TxRwdSet // optimistic code
	validationCode          peer.TxValidationCode
	containsPostOrderWrites bool
}

// publicAndHashUpdates encapsulates public and hash updates. The intended use of this to hold the updates
// that are to be applied to the statedb  as a result of the block commit
type publicAndHashUpdates struct {
	publicUpdates *privacyenabledstate.PubUpdateBatch
	hashUpdates   *privacyenabledstate.HashedUpdateBatch
	tempDbUpdates map[string]struct{}
}

// newPubAndHashUpdates constructs an empty PubAndHashUpdates
func newPubAndHashUpdates() *publicAndHashUpdates {
	return &publicAndHashUpdates{
		publicUpdates: privacyenabledstate.NewPubUpdateBatch(),
		hashUpdates:   privacyenabledstate.NewHashedUpdateBatch(),
		tempDbUpdates: make(map[string]struct{}),
	}
}

// containsPvtWrites returns true if this transaction is not limited to affecting the public data only
func (t *transaction) containsPvtWrites() bool {
	for _, ns := range t.rwdset.NsRwdSets {
		for _, coll := range ns.CollHashedRwSets {
			if coll.PvtRwSetHash != nil {
				return true
			}
		}
	}
	return false
}

// retrieveHash returns the hash of the private write-set present
// in the public data for a given namespace-collection
func (t *transaction) retrieveHash(ns string, coll string) []byte {
	if t.rwdset == nil {
		return nil
	}
	for _, nsData := range t.rwdset.NsRwdSets {
		if nsData.NameSpace != ns {
			continue
		}

		for _, collData := range nsData.CollHashedRwSets {
			if collData.CollectionName == coll {
				return collData.PvtRwSetHash
			}
		}
	}
	return nil
}

// applyWriteSet adds (or deletes) the key/values present in the write set to the publicAndHashUpdates
func (u *publicAndHashUpdates) applyWriteSet(
	txRWDSet *rwsetutil.TxRwdSet,
	txHeight *version.Height,
	db *privacyenabledstate.DB,
	containsPostOrderWrites bool,
) error {
	u.publicUpdates.ContainsPostOrderWrites =
		u.publicUpdates.ContainsPostOrderWrites || containsPostOrderWrites
	txops, err := prepareTxOps(txRWDSet, txHeight, u, db)
	logger.Debugf("txops=%#v", txops)
	if err != nil {
		return err
	}
	for compositeKey, keyops := range txops {
		if compositeKey.coll == "" {
			ns, key := compositeKey.ns, compositeKey.key
			if keyops.isDelete() {
				u.publicUpdates.Delete(ns, key, txHeight)
			} else {
				u.publicUpdates.PutValAndMetadata(ns, key, keyops.value, keyops.metadata, txHeight, false)
			}
		} else {
			ns, coll, keyHash := compositeKey.ns, compositeKey.coll, []byte(compositeKey.key)
			if keyops.isDelete() {
				u.hashUpdates.Delete(ns, coll, keyHash, txHeight)
			} else {
				u.hashUpdates.PutValHashAndMetadata(ns, coll, keyHash, keyops.value, keyops.metadata, txHeight)
			}
		}
	}
	return nil
}

// optimistic code begin
func (u *publicAndHashUpdates) applyWriteSetAndDeltaSet(
	txRWDSet *rwsetutil.TxRwdSet,
	txHeight *version.Height,
	db *privacyenabledstate.DB,
	containsPostOrderWrites bool,
) error {
	u.publicUpdates.ContainsPostOrderWrites =
		u.publicUpdates.ContainsPostOrderWrites || containsPostOrderWrites
	txops, err := prepareTxOps(txRWDSet, txHeight, u, db)
	logger.Debugf("txops=%#v", txops)
	if err != nil {
		return err
	}
	for compositeKey, keyops := range txops {
		if compositeKey.coll == "" {
			ns, key := compositeKey.ns, compositeKey.key
			if keyops.isDelete() {
				u.publicUpdates.Delete(ns, key, txHeight)
			} else {
				u.publicUpdates.PutValAndMetadata(ns, key, keyops.value, keyops.metadata, txHeight, false)
			}
		} else {
			ns, coll, keyHash := compositeKey.ns, compositeKey.coll, []byte(compositeKey.key)
			if keyops.isDelete() {
				u.hashUpdates.Delete(ns, coll, keyHash, txHeight)
			} else {
				u.hashUpdates.PutValHashAndMetadata(ns, coll, keyHash, keyops.value, keyops.metadata, txHeight)
			}
		}
	}
	// merge delta set into write set
	for _, nsRWDSet := range txRWDSet.NsRwdSets {
		ns := nsRWDSet.NameSpace
		for _, kvDelta := range nsRWDSet.KvRwdSet.Deltas {
			key := kvDelta.Key
			var payload []byte
			tempval := u.publicUpdates.Get(ns, key)
			if tempval == nil {
				// read from db
				dbval, err := db.GetState(ns, key)
				if err != nil {
					log.Fatalf("optimistic code FATAL, read from db key=%s", key)
				} else {
					payload = dbval.Value
				}
			} else {
				payload = tempval.Value
			}
			var verval ledger.VersionedValue
			err := json.Unmarshal(payload, &verval)
			if err != nil {
				log.Fatalf("unmarshal versionedvalue error: %v", err)
			}
			var delta_obj interface{}
			var db_obj interface{}
			err = json.Unmarshal(verval.Val, &db_obj)
			if err != nil {
				log.Fatalf("unmarshal versionedvalue error: %v", err)
			}
			err = json.Unmarshal(kvDelta.Value, &delta_obj)
			if err != nil {
				log.Fatalf("unmarshal versionedvalue error: %v", err)
			}
			t_db_obj, _ := db_obj.(map[string]interface{})
			t_delta_obj, _ := delta_obj.(map[string]interface{})
			for k, v := range t_delta_obj {
				switch vv := v.(type) {
				case int64:
					tempRes := t_db_obj[k].(int64) + vv
					t_delta_obj[k] = tempRes
				case float64:
					tempRes := t_db_obj[k].(float64) + vv
					t_delta_obj[k] = tempRes
					// case string:
					// log.Println("TODO string")
				}
			}
			verval.Val, err = json.Marshal(t_delta_obj)
			if err != nil {
				log.Fatalf("marshal verval when merge, error: %v", err)
			}
			verval.Txid = kvDelta.Txid
			bs, err := json.Marshal(verval)
			if err != nil {
				log.Fatalf("marshal verval when merge, error: %v", err)
			}
			// TODO: check metadata
			// log.Printf("optimistic merge delta %v", verval)
			u.publicUpdates.Put(ns, key, bs, txHeight, true)
		}
	}
	return nil
}

// optimistic code end

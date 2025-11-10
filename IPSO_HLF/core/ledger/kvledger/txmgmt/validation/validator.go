//go:build !crdt && !preread
// +build !crdt,!preread

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validation

import (
	"encoding/json"
	"log"

	"github.com/PanQL/fabric-protos-go/ledger/rwset/kvrwset"
	"github.com/PanQL/fabric-protos-go/peer"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger/internal/version"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger/kvledger/txmgmt/statedb"
	"github.com/davecgh/go-spew/spew"
)

// optimistic code begin
type VerVal struct {
	Txid string
	Val  []byte
}

// optimistic code end

// validator validates a tx against the latest committed state
// and preceding valid transactions with in the same block
type validator struct {
	db       *privacyenabledstate.DB
	hashFunc rwsetutil.HashFunc

	// benchmark
	preread_success_intra int
	preread_success_inter int
	preread_fail          int
	intra_aborts          int
	inter_aborts          int
}

// preLoadCommittedVersionOfRSet loads committed version of all keys in each
// transaction's read set into a cache.
func (v *validator) preLoadCommittedVersionOfRSet(blk *block) error {

	// Collect both public and hashed keys in read sets of all transactions in a given block
	var pubKeys []*statedb.CompositeKey
	var hashedKeys []*privacyenabledstate.HashedCompositeKey

	// pubKeysMap and hashedKeysMap are used to avoid duplicate entries in the
	// pubKeys and hashedKeys. Though map alone can be used to collect keys in
	// read sets and pass as an argument in LoadCommittedVersionOfPubAndHashedKeys(),
	// array is used for better code readability. On the negative side, this approach
	// might use some extra memory.
	pubKeysMap := make(map[statedb.CompositeKey]interface{})
	hashedKeysMap := make(map[privacyenabledstate.HashedCompositeKey]interface{})

	for _, tx := range blk.txs {
		for _, nsRWSet := range tx.rwdset.NsRwdSets {
			for _, kvRead := range nsRWSet.KvRwdSet.Reads {
				compositeKey := statedb.CompositeKey{
					Namespace: nsRWSet.NameSpace,
					Key:       kvRead.Key,
				}
				if _, ok := pubKeysMap[compositeKey]; !ok {
					pubKeysMap[compositeKey] = nil
					pubKeys = append(pubKeys, &compositeKey)
				}

			}
			for _, colHashedRwSet := range nsRWSet.CollHashedRwSets {
				for _, kvHashedRead := range colHashedRwSet.HashedRwSet.HashedReads {
					hashedCompositeKey := privacyenabledstate.HashedCompositeKey{
						Namespace:      nsRWSet.NameSpace,
						CollectionName: colHashedRwSet.CollectionName,
						KeyHash:        string(kvHashedRead.KeyHash),
					}
					if _, ok := hashedKeysMap[hashedCompositeKey]; !ok {
						hashedKeysMap[hashedCompositeKey] = nil
						hashedKeys = append(hashedKeys, &hashedCompositeKey)
					}
				}
			}
		}
	}

	// Load committed version of all keys into a cache
	if len(pubKeys) > 0 || len(hashedKeys) > 0 {
		err := v.db.LoadCommittedVersionsOfPubAndHashedKeys(pubKeys, hashedKeys)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateAndPrepareBatch performs validation and prepares the batch for final writes
func (v *validator) validateAndPrepareBatch(blk *block, doMVCCValidation bool) (*publicAndHashUpdates, error) {
	v.preread_fail = 0
	v.preread_success_intra = 0
	v.preread_success_inter = 0
	v.inter_aborts = 0
	v.intra_aborts = 0
	defer func() {
		logger.Infof("benchmark micro aborts: intra_aborts %d inter_aborts %d preread_success_intra %d preread_success_inter %d preread_fail %d", v.intra_aborts, v.inter_aborts, v.preread_success_intra, v.preread_success_inter, v.preread_fail)

	}()

	// Check whether statedb implements BulkOptimizable interface. For now,
	// only CouchDB implements BulkOptimizable to reduce the number of REST
	// API calls from peer to CouchDB instance.
	if v.db.IsBulkOptimizable() {
		err := v.preLoadCommittedVersionOfRSet(blk)
		if err != nil {
			return nil, err
		}
	}
	// // debug
	// logger.Infof("\ndebug v4 blockid=%d txns=%d\n", blk.num, len(blk.txs))

	// for _, tx := range blk.txs {
	// 	temp := fmt.Sprintf("txid=%s idInBlk=%d\n", tx.id, tx.indexInBlock)
	// 	for _, nsRWSSet := range tx.rwdset.NsRwdSets[1:] { // drop lifecycle
	// 		temp += fmt.Sprintf("\tnamespace=%s\n", nsRWSSet.NameSpace)
	// 		temp += fmt.Sprintf("\t\treads\n")
	// 		for _, kvread := range nsRWSSet.KvRwdSet.Reads {
	// 			temp += fmt.Sprintf("\t\t\tkey=%s version=%v, txid=%s\n", kvread.Key, kvread.Version, kvread.Txid)
	// 		}
	// 		temp += fmt.Sprintf("\t\twrites\n")
	// 		for _, kvwrite := range nsRWSSet.KvRwdSet.Writes {
	// 			temp += fmt.Sprintf("\t\t\tkey=%s\n", kvwrite.Key)
	// 		}
	// 		temp += fmt.Sprintf("\t\tdeltas\n")
	// 		for _, kvdelta := range nsRWSSet.KvRwdSet.Deltas {
	// 			temp += fmt.Sprintf("\t\t\tkey=%s txid=%s", kvdelta.Key, kvdelta.Txid)
	// 		}
	// 	}
	// 	logger.Info(temp)
	// }

	updates := newPubAndHashUpdates()
	for _, tx := range blk.txs {
		var validationCode peer.TxValidationCode
		var err error
		if validationCode, err = v.validateEndorserTX(tx.rwdset, doMVCCValidation, updates); err != nil {
			return nil, err
		}

		tx.validationCode = validationCode
		if validationCode == peer.TxValidationCode_VALID {
			logger.Debugf("Block [%d] Transaction index [%d] TxId [%s] marked as valid by state validator. ContainsPostOrderWrites [%t]", blk.num, tx.indexInBlock, tx.id, tx.containsPostOrderWrites)
			committingTxHeight := version.NewHeight(blk.num, uint64(tx.indexInBlock))
			if err := updates.applyWriteSetAndDeltaSet(tx.rwdset, committingTxHeight, v.db, tx.containsPostOrderWrites); err != nil { // optimistic code
				return nil, err
			}
		} else {
			logger.Warningf("Block [%d] Transaction index [%d] TxId [%s] marked as invalid by state validator. Reason code [%s]",
				blk.num, tx.indexInBlock, tx.id, validationCode.String())
		}
	}
	return updates, nil
}

// validateEndorserTX validates endorser transaction
func (v *validator) validateEndorserTX(
	txRWDSet *rwsetutil.TxRwdSet,
	doMVCCValidation bool,
	updates *publicAndHashUpdates) (peer.TxValidationCode, error) {

	var validationCode = peer.TxValidationCode_VALID
	var err error
	//mvcc validation, may invalidate transaction
	if doMVCCValidation {
		validationCode, err = v.validateTx(txRWDSet, updates)
	}
	return validationCode, err
}

func (v *validator) validateTx(txRWDSet *rwsetutil.TxRwdSet, updates *publicAndHashUpdates) (peer.TxValidationCode, error) {
	// Uncomment the following only for local debugging. Don't want to print data in the logs in production
	logger.Debugf("validateTx - validating txRWDSet: %s", spew.Sdump(txRWDSet))
	for _, nsRWDSet := range txRWDSet.NsRwdSets {
		ns := nsRWDSet.NameSpace
		// Validate public reads
		if valid, err := v.validateReadSet(ns, nsRWDSet.KvRwdSet.Reads, updates.publicUpdates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			for _, kvRead := range nsRWDSet.KvRwdSet.Reads {
				updates.tempDbUpdates[kvRead.Key] = struct{}{}
			}
			return peer.TxValidationCode_MVCC_READ_CONFLICT, nil
		}
		// PandoTODO: validate path conditions
		if valid, err := v.validatePathCond(ns, nsRWDSet.KvRwdSet.Condition, updates.publicUpdates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_MVCC_READ_CONFLICT, nil // PandoTODO: change to HOT_PATH_COND_VIOLATED
		}
		// Validate range queries for phantom items
		if valid, err := v.validateRangeQueries(ns, nsRWDSet.KvRwdSet.RangeQueriesInfo, updates.publicUpdates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_PHANTOM_READ_CONFLICT, nil
		}
		// Validate hashes for private reads
		if valid, err := v.validateNsHashedReadSets(ns, nsRWDSet.CollHashedRwSets, updates.hashUpdates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_MVCC_READ_CONFLICT, nil
		}
	}
	return peer.TxValidationCode_VALID, nil
}

func compareAccountBalanceLarger(balance interface{}, amount int64) bool {
	switch tt := balance.(type) {
	case int:
		return tt > int(amount)
	case float64:
		return tt > float64(amount)
	default:
		// Unsupported type for balance, hence return false
		return false
	}
}

func compareAccountBalanceLargerEqual(balance interface{}, amount int64) bool {
	switch tt := balance.(type) {
	case int:
		return tt >= int(amount)
	case float64:
		return tt >= float64(amount)
	default:
		// Unsupported type for balance, hence return false
		return false
	}
}

func validateAccountCondition(account map[string]interface{}, pathCond *peer.HotPathCond) (bool, error) {
	switch pathCond.CondType {
	case peer.HotPathCondType_CHECKING_LARGER:
		balance := account["checkingBalance"]
		return compareAccountBalanceLarger(balance, pathCond.Amount), nil
	case peer.HotPathCondType_CHECKING_LARGER_EQUAL:
		balance := account["checkingBalance"]
		return compareAccountBalanceLargerEqual(balance, pathCond.Amount), nil
	case peer.HotPathCondType_SAVING_LARGER:
		balance := account["savingsBalance"]
		return compareAccountBalanceLarger(balance, pathCond.Amount), nil
	case peer.HotPathCondType_SAVING_LARGER_EQUAL:
		balance := account["savingsBalance"]
		return compareAccountBalanceLargerEqual(balance, pathCond.Amount), nil
	default:
		// Unsupported path condition type, hence return false
		return false, nil
	}
}

func (v *validator) validatePathCond(ns string, pathCond *peer.HotPathCond, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	var version VerVal

	if pathCond == nil {
		return true, nil // no path condition, hence return true
	}

	if updates.Exists(ns, pathCond.Key) {
		vv := updates.Get(ns, pathCond.Key)
		err := json.Unmarshal(vv.Value, &version)
		if err != nil {
			log.Fatalln("please check versionedValue in validataeKVRead", err)
		}
	} else {
		vv, err := v.db.GetState(ns, pathCond.Key)
		if err != nil {
			// which case?
			return false, err
		}
		if vv == nil {
			// Key does not exist in the state, hence path condition is violated
			return false, nil
		}
		err = json.Unmarshal(vv.Value, &version)
		if err != nil {
			log.Fatalln("please check versionedValue in validataeKVRead", err)
		}
	}

	var account map[string]interface{}
	if err := json.Unmarshal(version.Val, &account); err != nil {
		log.Fatalln("please check account in validataeKVRead", err)
	}
	return validateAccountCondition(account, pathCond)
}

// //////////////////////////////////////////////////////////////////////////////
// ///                 Validation of public read-set
// //////////////////////////////////////////////////////////////////////////////
func (v *validator) validateReadSet(ns string, kvReads []*kvrwset.KVRead, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	for _, kvRead := range kvReads {
		if valid, err := v.validateKVRead(ns, kvRead, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// validateKVRead performs mvcc check for a key read during transaction simulation.
// i.e., it checks whether a key/version combination is already updated in the statedb (by an already committed block)
// or in the updates (by a preceding valid transaction in the current block)
func (v *validator) validateKVRead(ns string, kvRead *kvrwset.KVRead, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	if updates.Exists(ns, kvRead.Key) {
		// optimistic code begin
		v.intra_aborts += 1
		vv := updates.Get(ns, kvRead.Key)
		var verval VerVal
		err := json.Unmarshal(vv.Value, &verval)
		if err != nil {
			log.Fatalln("please check VersionedValue if value=txid+value", err)
		}
		if kvRead.Txid == verval.Txid {
			// log.Println("same with previous uncommitted key")
			v.preread_success_intra += 1
			return true, nil
		} else {
			// log.Println("not same with previous uncommitted key")
			v.preread_fail += 1
			return false, nil
		}
		// return false, nil
		// optimistic code end
	}
	versionedValue, err := v.db.GetState(ns, kvRead.Key)
	if err != nil {
		// which case?
		v.preread_fail += 1
		return false, err
	}
	var committedVersion *version.Height
	if versionedValue == nil {
		committedVersion = nil
	} else {
		committedVersion = versionedValue.Version
	}

	logger.Debugf("Comparing versions for key [%s]: committed version=%#v and read version=%#v",
		kvRead.Key, committedVersion, rwsetutil.NewVersion(kvRead.Version))
	if !version.AreSame(committedVersion, rwsetutil.NewVersion(kvRead.Version)) {
		// optimistic code begin
		v.inter_aborts += 1
		var version VerVal
		err := json.Unmarshal(versionedValue.Value, &version)
		if err != nil {
			log.Fatalln("please check versionedValue in validataeKVRead", err)
		}
		if version.Txid == kvRead.Txid {
			v.preread_success_inter += 1
			return true, nil
		} else {
			logger.Debugf("Version mismatch for key [%s:%s] and txid [%s:%s]. Committed version = [%#v], Version in readSet [%#v]",
				ns, kvRead.Key, version.Txid, kvRead.Txid, committedVersion, kvRead.Version)
			return false, nil
		}
		// logger.Debugf("Version mismatch for key [%s:%s]. Committed version = [%#v], Version in readSet [%#v]",
		// 	ns, kvRead.Key, committedVersion, kvRead.Version)
		// return false, nil

		// optimistic code end
	}
	return true, nil
}

// //////////////////////////////////////////////////////////////////////////////
// ///                 Validation of range queries
// //////////////////////////////////////////////////////////////////////////////
func (v *validator) validateRangeQueries(ns string, rangeQueriesInfo []*kvrwset.RangeQueryInfo, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	for _, rqi := range rangeQueriesInfo {
		if valid, err := v.validateRangeQuery(ns, rqi, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// validateRangeQuery performs a phantom read check i.e., it
// checks whether the results of the range query are still the same when executed on the
// statedb (latest state as of last committed block) + updates (prepared by the writes of preceding valid transactions
// in the current block and yet to be committed as part of group commit at the end of the validation of the block)
func (v *validator) validateRangeQuery(ns string, rangeQueryInfo *kvrwset.RangeQueryInfo, updates *privacyenabledstate.PubUpdateBatch) (bool, error) {
	logger.Debugf("validateRangeQuery: ns=%s, rangeQueryInfo=%s", ns, rangeQueryInfo)

	// If during simulation, the caller had not exhausted the iterator so
	// rangeQueryInfo.EndKey is not actual endKey given by the caller in the range query
	// but rather it is the last key seen by the caller and hence the combinedItr should include the endKey in the results.
	includeEndKey := !rangeQueryInfo.ItrExhausted

	combinedItr, err := newCombinedIterator(v.db, updates.UpdateBatch,
		ns, rangeQueryInfo.StartKey, rangeQueryInfo.EndKey, includeEndKey)
	if err != nil {
		return false, err
	}
	defer combinedItr.Close()
	var qv rangeQueryValidator
	if rangeQueryInfo.GetReadsMerkleHashes() != nil {
		logger.Debug(`Hashing results are present in the range query info hence, initiating hashing based validation`)
		qv = &rangeQueryHashValidator{hashFunc: v.hashFunc}
	} else {
		logger.Debug(`Hashing results are not present in the range query info hence, initiating raw KVReads based validation`)
		qv = &rangeQueryResultsValidator{}
	}
	if err := qv.init(rangeQueryInfo, combinedItr); err != nil {
		return false, err
	}
	return qv.validate()
}

// //////////////////////////////////////////////////////////////////////////////
// ///                 Validation of hashed read-set
// //////////////////////////////////////////////////////////////////////////////
func (v *validator) validateNsHashedReadSets(ns string, collHashedRWSets []*rwsetutil.CollHashedRwSet,
	updates *privacyenabledstate.HashedUpdateBatch) (bool, error) {
	for _, collHashedRWSet := range collHashedRWSets {
		if valid, err := v.validateCollHashedReadSet(ns, collHashedRWSet.CollectionName, collHashedRWSet.HashedRwSet.HashedReads, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

func (v *validator) validateCollHashedReadSet(ns, coll string, kvReadHashes []*kvrwset.KVReadHash,
	updates *privacyenabledstate.HashedUpdateBatch) (bool, error) {
	for _, kvReadHash := range kvReadHashes {
		if valid, err := v.validateKVReadHash(ns, coll, kvReadHash, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// validateKVReadHash performs mvcc check for a hash of a key that is present in the private data space
// i.e., it checks whether a key/version combination is already updated in the statedb (by an already committed block)
// or in the updates (by a preceding valid transaction in the current block)
func (v *validator) validateKVReadHash(ns, coll string, kvReadHash *kvrwset.KVReadHash,
	updates *privacyenabledstate.HashedUpdateBatch) (bool, error) {
	if updates.Contains(ns, coll, kvReadHash.KeyHash) {
		return false, nil
	}
	committedVersion, err := v.db.GetKeyHashVersion(ns, coll, kvReadHash.KeyHash)
	if err != nil {
		return false, err
	}

	if !version.AreSame(committedVersion, rwsetutil.NewVersion(kvReadHash.Version)) {
		logger.Debugf("Version mismatch for key hash [%s:%s:%#v]. Committed version = [%s], Version in hashedReadSet [%s]",
			ns, coll, kvReadHash.KeyHash, committedVersion, kvReadHash.Version)
		return false, nil
	}
	return true, nil
}

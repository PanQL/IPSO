/*
Optimistic code
*/
package rwsetutil

import (
	"github.com/PanQL/fabric-protos-go/ledger/rwset"
	"github.com/PanQL/fabric-protos-go/ledger/rwset/kvrwset"
	"github.com/golang/protobuf/proto"
)

/////////////////////////////////////////////////////////////////
// Messages related to PUBLIC read-write set
/////////////////////////////////////////////////////////////////

// TxRwdSet acts as a proxy of 'rwset.TxReadWriteDeltaSet' proto message and helps constructing Read-write set specifically for KV data model
type TxRwdSet struct {
	NsRwdSets []*NsRwdSet
}

// NsRwdSet encapsulates 'kvrwset.KVRWDSet' proto message for a specific name space (chaincode)
type NsRwdSet struct {
	NameSpace        string
	KvRwdSet         *kvrwset.KVRWDSet
	CollHashedRwSets []*CollHashedRwSet
}

// GetPvtDataHash returns the PvtRwSetHash for a given namespace and collection
func (txRwdSet *TxRwdSet) GetPvtDataHash(ns, coll string) []byte {
	// we could build and use a map to reduce the number of lookup
	// in the future call. However, we decided to defer such optimization
	// due to the following assumptions (mainly to avoid additional LOC).
	// we assume that the number of namespaces and collections in a txRWSet
	// to be very minimal (in a single digit),
	for _, nsRwdSet := range txRwdSet.NsRwdSets {
		if nsRwdSet.NameSpace != ns {
			continue
		}
		return nsRwdSet.getPvtDataHash(coll)
	}
	return nil
}

func (nsRwdSet *NsRwdSet) getPvtDataHash(coll string) []byte {
	for _, collHashedRwSet := range nsRwdSet.CollHashedRwSets {
		if collHashedRwSet.CollectionName != coll {
			continue
		}
		return collHashedRwSet.PvtRwSetHash
	}
	return nil
}

/////////////////////////////////////////////////////////////////
// FUNCTIONS for converting messages to/from proto bytes
/////////////////////////////////////////////////////////////////

// ToProtoBytes constructs TxReadWriteSet proto message and serializes using protobuf Marshal
func (txRwdSet *TxRwdSet) ToProtoBytes() ([]byte, error) {
	var protoMsg *rwset.TxReadWriteDeltaSet
	var err error
	if protoMsg, err = txRwdSet.toProtoMsg(); err != nil {
		return nil, err
	}
	return proto.Marshal(protoMsg)
}

// FromProtoBytes deserializes protobytes into TxReadWriteSet proto message and populates 'TxRwdSet'
func (txRwdSet *TxRwdSet) FromProtoBytes(protoBytes []byte) error {
	protoMsg := &rwset.TxReadWriteDeltaSet{}
	var err error
	var txRwSetTemp *TxRwdSet
	if err = proto.Unmarshal(protoBytes, protoMsg); err != nil {
		return err
	}
	if txRwSetTemp, err = TxRwdSetFromProtoMsg(protoMsg); err != nil {
		return err
	}
	txRwdSet.NsRwdSets = txRwSetTemp.NsRwdSets
	return nil
}

func (txRwdSet *TxRwdSet) toProtoMsg() (*rwset.TxReadWriteDeltaSet, error) {
	protoMsg := &rwset.TxReadWriteDeltaSet{DataModel: rwset.TxReadWriteDeltaSet_KV}
	var nsRwdSetProtoMsg *rwset.NsReadWriteDeltaSet
	var err error
	for _, nsRwdSet := range txRwdSet.NsRwdSets {
		if nsRwdSetProtoMsg, err = nsRwdSet.toProtoMsg(); err != nil {
			return nil, err
		}
		protoMsg.NsRwdset = append(protoMsg.NsRwdset, nsRwdSetProtoMsg)
	}
	return protoMsg, nil
}

func TxRwdSetFromProtoMsg(protoMsg *rwset.TxReadWriteDeltaSet) (*TxRwdSet, error) {
	txRwdSet := &TxRwdSet{}
	var nsRwdSet *NsRwdSet
	var err error
	for _, nsRwdSetProtoMsg := range protoMsg.NsRwdset {
		if nsRwdSet, err = nsRwdSetFromProtoMsg(nsRwdSetProtoMsg); err != nil {
			return nil, err
		}
		txRwdSet.NsRwdSets = append(txRwdSet.NsRwdSets, nsRwdSet)
	}
	return txRwdSet, nil
}

func (nsRwdSet *NsRwdSet) toProtoMsg() (*rwset.NsReadWriteDeltaSet, error) {
	var err error
	protoMsg := &rwset.NsReadWriteDeltaSet{Namespace: nsRwdSet.NameSpace}
	if protoMsg.Rwdset, err = proto.Marshal(nsRwdSet.KvRwdSet); err != nil {
		return nil, err
	}

	var collHashedRwSetProtoMsg *rwset.CollectionHashedReadWriteSet
	for _, collHashedRwSet := range nsRwdSet.CollHashedRwSets {
		if collHashedRwSetProtoMsg, err = collHashedRwSet.toProtoMsg(); err != nil {
			return nil, err
		}
		protoMsg.CollectionHashedRwset = append(protoMsg.CollectionHashedRwset, collHashedRwSetProtoMsg)
	}
	return protoMsg, nil
}

func nsRwdSetFromProtoMsg(protoMsg *rwset.NsReadWriteDeltaSet) (*NsRwdSet, error) {
	nsRwSet := &NsRwdSet{NameSpace: protoMsg.Namespace, KvRwdSet: &kvrwset.KVRWDSet{}}
	if err := proto.Unmarshal(protoMsg.Rwdset, nsRwSet.KvRwdSet); err != nil {
		return nil, err
	}
	var err error
	var collHashedRwSet *CollHashedRwSet
	for _, collHashedRwSetProtoMsg := range protoMsg.CollectionHashedRwset {
		if collHashedRwSet, err = collHashedRwSetFromProtoMsg(collHashedRwSetProtoMsg); err != nil {
			return nil, err
		}
		nsRwSet.CollHashedRwSets = append(nsRwSet.CollHashedRwSets, collHashedRwSet)
	}
	return nsRwSet, nil
}

func (txRwdSet *TxRwdSet) NumCollections() int {
	if txRwdSet == nil {
		return 0
	}
	numColls := 0
	for _, nsRwdset := range txRwdSet.NsRwdSets {
		for range nsRwdset.CollHashedRwSets {
			numColls++
		}
	}
	return numColls
}

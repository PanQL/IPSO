package txmgr

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Yunpeng-J/HLF-2.2/core/ledger"
	"github.com/Yunpeng-J/HLF-2.2/core/ledger/kvledger/txmgmt/rwsetutil"
)

type TempDB struct {
	cmtOrderGen int32
	// mutex       sync.Mutex
	rwLock     sync.RWMutex
	Sessions   map[string]*SessionDB
	KeySession map[string]string
}

type SessionDB struct {
	session string
	db      map[string][]byte // key is a string
	tempdb  *TempDB
}

func NewTempDB() *TempDB {
	return newTempDB()
}

func newSessionDB(session string, tdb *TempDB) *SessionDB {
	return &SessionDB{
		session: session,
		db:      map[string][]byte{},
		tempdb:  tdb,
	}
}

func (sdb *SessionDB) Get(key string) (*ledger.VersionedValue, []byte) {
	val, ok := sdb.db[key]
	if !ok {
		return nil, nil
	}

	versionedValue := &ledger.VersionedValue{}
	err := json.Unmarshal(val, versionedValue)
	if err != nil {
		log.Fatalln("get from session db,", key, versionedValue.Txid, err)
	}
	return versionedValue, val
}

func (sdb *SessionDB) Delete(key string) {
	delete(sdb.db, key)
}

func (sdb *SessionDB) Commit(txid string, rwdSet *rwsetutil.TxRwdSet) {
	for _, rwd := range rwdSet.NsRwdSets {
		if rwd.NameSpace == "smallbank" {
			for _, kvwrite := range rwd.KvRwdSet.Writes {
				sdb.db[kvwrite.Key] = kvwrite.Value
				sdb.tempdb.KeySession[kvwrite.Key] = sdb.session
			}
			break
		}
	}
}

func newTempDB() *TempDB {
	res := &TempDB{
		Sessions:    map[string]*SessionDB{},
		KeySession:  map[string]string{},
		cmtOrderGen: 0,
	}
	return res
}

func (tdb *TempDB) Get(key, session string) (*ledger.VersionedValue, []byte) {
	if session == "" {
		return nil, nil
	}
	tdb.rwLock.RLock()
	defer tdb.rwLock.RUnlock()
	// tdb.mutex.Lock()
	// defer tdb.mutex.Unlock()
	if tdb.KeySession[key] != session {
		// TODO: clean
		return nil, nil
	}
	sdb, ok := tdb.Sessions[session]
	if ok {
		return sdb.Get(key)
	} else {
		return nil, nil
	}
}

func (tdb *TempDB) Put(key, txid string, val []byte) {
	// // val has already been encoded by marshaling (txid, ori_val)
	// session := GetSessionFromTxid(txid)
	// if session == "" {
	// 	return
	// }
	// tdb.mutex.Lock()
	// sdb, ok := tdb.Sessions[session]
	// if ok {
	// 	tdb.mutex.Unlock()
	// 	sdb.Put(txid, key, val)
	// } else {
	// 	db := newSessionDB(session)
	// 	tdb.Sessions[session] = db
	// 	tdb.mutex.Unlock()
	// 	db.Put(txid, key, val)
	// }
}

func (tdb *TempDB) Rollback(txid string) {
	// session := GetSessionFromTxid(txid)
	// if session == "" {
	// 	return
	// }
	// tdb.mutex.Lock()
	// sdb, ok := tdb.Sessions[session]
	// tdb.mutex.Unlock()
	// if ok {
	// 	sdb.Rollback(txid)
	// } else {
	// 	log.Fatalln("something is wrong with the Temp DB, please check it")
	// }
}

func (tdb *TempDB) Commit(txid string, rwdSet *rwsetutil.TxRwdSet) int32 {
	session := GetSessionFromTxid(txid)
	if session == "" {
		return -1
	}
	tdb.rwLock.Lock()
	defer tdb.rwLock.Unlock()
	// tdb.mutex.Lock()
	sdb, ok := tdb.Sessions[session]
	// defer tdb.mutex.Unlock()
	if ok {
		sdb.Commit(txid, rwdSet)
	} else {
		sdb = newSessionDB(session, tdb)
		tdb.Sessions[session] = sdb
		sdb.Commit(txid, rwdSet)
	}
	order := tdb.cmtOrderGen
	tdb.cmtOrderGen++
	return order
}

// Prune: delete all obsolete keys
func (tdb *TempDB) Prune(deltaSet *map[string]*ledger.VersionedValue) {
	// TODO: optimization
	st := time.Now()
	defer func() {
		log.Printf("benchmark prune tempdb with %d keys in %d ms\n", len(*deltaSet), time.Since(st).Milliseconds())
	}()
	tdb.rwLock.Lock()
	defer tdb.rwLock.Unlock()
	// tdb.mutex.Lock()
	// defer tdb.mutex.Unlock()
	for sname, _ := range tdb.Sessions {
		delete(tdb.Sessions, sname)
	}
	// for k, s := range *deltaSet {
	// 	session := GetSessionFromTxid(s.Txid)
	// 	tdb.KeySession[k] = session
	// 	// 80~100 ms
	// 	for sname, sdb := range tdb.Sessions {
	// 		if sname != session {
	// 			sdb.Delete(k)
	// 		}
	// 	}
	// }
}

func (tdb *TempDB) PruneKeys(deltaSet *map[string]struct{}) {
	tdb.rwLock.Lock()
	defer tdb.rwLock.Unlock()
	// tdb.mutex.Lock()
	// defer tdb.mutex.Unlock()
	for k, _ := range *deltaSet {
		for _, sdb := range tdb.Sessions {
			sdb.Delete(k)
		}
	}
}

// func (tdb *TempDB) String() string {
// 	var res string
// 	tdb.mutex.Lock()
// 	defer tdb.mutex.Unlock()
// 	for session, sessiondb := range tdb.Sessions {
// 		res += fmt.Sprintf("session:%s\n", session)
// 		res += fmt.Sprintf("\tcommitdb:\n")
// 		for key, val := range sessiondb.db {
// 			var verval ledger.VersionedValue
// 			err := json.Unmarshal(val, &verval)
// 			if err != nil {
// 				log.Fatalf("stringfy tempdb, unmarshal error: %v", err)

// 			}
// 			res += fmt.Sprintf("\t\tkey=%s; txid=%s, val=%s\n", key, verval.Txid, verval.Val)
// 		}
// 		res += fmt.Sprintf("\tuncommitdb:\n")
// 		// for txid, rws := range sessiondb.writeSets {
// 		// 	res += fmt.Sprintf("\t\ttxid=%s:\n", txid)
// 		// 	for i := 0; i < len(rws.keys); i++ {
// 		// 		res += fmt.Sprintf("\t\t\tkey=%s; val=%s\n", rws.keys[i], string(rws.vals[i]))
// 		// 	}
// 		// }
// 	}
// 	res += fmt.Sprintf("\n")
// 	return res
// }

// txid format: seqNumber_Session_oriTxid
func GetSessionFromTxid(txid string) string {
	temp := strings.Split(txid, "_+=+_")
	if len(temp) == 1 {
		return ""
	} else if len(temp) == 3 {
		return temp[1]
	}
	return ""
}

func GetSeqFromTxid(txid string) string {
	temp := strings.Split(txid, "_+=+_")
	if len(temp) == 1 {
		return ""
	} else if len(temp) == 3 {
		return temp[0]
	}
	return ""
}

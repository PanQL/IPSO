package chaincode_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/Yunpeng-J/HLF-2.2/core/chaincode"
)

func comtxid(seq, session, txid int) string {
	return strconv.Itoa(seq) + "_+=+_" + strconv.Itoa(session) + "_+=+_" + strconv.Itoa(txid)
}

func Test1(t *testing.T) {
	txid := comtxid(2, 45, 23)
	res := chaincode.GetSessionFromTxid(txid)
	if res != "45" {
		t.Fatalf("expect 45, but got %s", res)
	}
	res = chaincode.GetSeqFromTxid(txid)
	if res != "2" {
		t.Fatalf("expect 2, but got %s", res)
	}
}

func Test2(t *testing.T) {
	db := chaincode.NewTempDB()
	orival := []byte("val1")
	txid := comtxid(0, 1, 1)
	verval := &chaincode.VersionedValue{
		Txid: txid,
		Val:  orival,
	}
	valtxid, err := json.Marshal(verval)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	db.Put("key1", txid, valtxid)
	db.Put("key2", comtxid(2, 1, 234), []byte("session=1 key=key2 val=234"))
	db.Put("key2", comtxid(2, 123, 234), []byte("session=123"))
	val := db.Get("key1", "1")
	if val != nil {
		t.Fatalf("expect nil, but got %v", val)
	}
	fmt.Println(db)
	db.Commit(txid)
	val = db.Get("key1", "1")
	if val.Txid != txid {
		t.Fatalf("expect %v, but got %v", txid, val.Txid)
	}
	if !bytes.Equal(val.Val, []byte("val1")) {
		t.Fatalf("expect %v, but got %v", []byte("val1"), val.Val)
	}
	fmt.Printf("commit key1\n%v", db)
	db.Prune(txid)
	fmt.Printf("prune %s\n%v", txid, db)
	db.Put("key1", txid, valtxid)
	fmt.Printf("put key1\n%v", db)
	db.Rollback(txid)
	fmt.Printf("rollback key1\n%v", db)
}

// var _ = Describe("tempdb", func() {
// 	var (
// 		db *chaincode.TempDB
// 	)
//
// 	BeforeEach(func() {
// 		db = chaincode.NewTempDB()
// 	})
//
// 	Describe("commit", func() {
// 		Context("put", func() {
// 			It("commit", func() {
// 				db.Put("key1", comtxid(0, 1, 1), []byte("val1"))
// 				Expect(db.Get("key1", "1")).To(Equal([]byte("val1")))
// 			})
// 		})
// 	})
//
// })

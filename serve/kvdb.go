package serve

import (
	"fmt"
	"github.com/HouzuoGuo/tiedot/db"
	"github.com/hidu/goutils"
	"log"
	"time"
)

type TieDb struct {
	tdb           *db.DB
	RequestTable  *KvTable
	ResponseTable *KvTable
}

type KvTable struct {
	name string
	col  *db.Col
}

func newTable(mydb *db.DB, name string) *KvTable {
	if err := mydb.Create(name); err != nil {
		log.Println(err)
	}
	mydb.Scrub(name)
	myTable := mydb.Use(name)
	return &KvTable{name: name, col: myTable}
}

func (tb *KvTable) GetByKey(id int) (data kvType, err error) {
	data, err = tb.col.Read(id)
	if err == nil {
		data["id"] = fmt.Sprintf("%d", id)
	}
	return
}

func (tb *KvTable) Set(id int, data kvType) error {
	return tb.col.InsertRecovery(id, data)
}

func NewTieDb(dir string) *TieDb {
	mydb, err := db.OpenDB(dir)
	if err != nil {
		panic(err)
	}
	tdb := &TieDb{
		tdb:           mydb,
		RequestTable:  newTable(mydb, "req"),
		ResponseTable: newTable(mydb, "res"),
	}
	return tdb
}

func (t *TieDb) Flush() {
	t.tdb.Sync()
}

func (t *TieDb) Clean(max_time_unix int64) {
	//	t.RequestTable.ForEachDoc(func(id int, data []byte) bool {
	//		if int64(doc["now"].(float64)) < max_time_unix {
	//			t.RequestTable.Delete(id)
	//			log.Println("delete expire req,", id)
	//		}
	//		return true
	//	})
	//	t.ResponseTable.ForEachDoc(func(id int, data []byte) bool {
	//		if int64(doc["now"].(float64)) < max_time_unix {
	//			t.RequestTable.Delete(id)
	//			log.Println("delete expire res,", id)
	//		}
	//		return true
	//	})
}

func (t *TieDb) StartGcTimer(sec int64, max_life int64) {
	utils.SetInterval(func() {
		t.Clean(time.Now().Unix() - max_life)
	}, sec)
}

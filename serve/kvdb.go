package serve

import (
	"encoding/json"
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
	tdb  *db.DB
}

func newTable(mydb *db.DB, name string) *KvTable {
	if err := mydb.Create(name); err != nil {
		log.Println(err)
	}
	//	mydb.Scrub(name)
	myTable := mydb.Use(name)
	return &KvTable{name: name, col: myTable, tdb: mydb}
}

func (tb *KvTable) GetByKey(id int) (data KvType, err error) {
	data, err = tb.col.Read(id)
	if err == nil {
		data["id"] = fmt.Sprintf("%d", id)
	}
	return
}
func (tb *KvTable) Scrub() {
	tb.tdb.Scrub(tb.name)
	tb.col = tb.tdb.Use(tb.name)
}

func (tb *KvTable) Set(id int, data KvType) error {

	if _, has := data["now"]; !has {
		data["now"] = time.Now().Unix()
	}
	return tb.col.InsertRecovery(id, data)
}

func (tb *KvTable) Delete(id int) error {
	err := tb.col.Delete(id)
	log.Println("delete [", tb.name, "] [", id, "]", err)
	return err
}

func (tb *KvTable) ForEachDoc(fn func(int, []byte) bool) {
	tb.col.ForEachDoc(fn)
}

func (tb *KvTable) Count() int {
	return tb.col.ApproxDocCount()
}
func (tb *KvTable) Gc(max_time_unix int64) int {
	deleteIds := make([]int, 0, 10000)
	tb.ForEachDoc(func(id int, data []byte) bool {
		var doc KvType
		if err := json.Unmarshal([]byte(data), &doc); err != nil {
			deleteIds = append(deleteIds, id)
			return true
		}
		_, hasNow := doc["now"]
		if (hasNow && int64(doc["now"].(float64)) < max_time_unix) || !hasNow {
			deleteIds = append(deleteIds, id)
		}
		return true

	})
	n := len(deleteIds)
	log.Println("table gc [", tb.name, "] total:", n)

	for _, id := range deleteIds {
		tb.Delete(id)
	}
	return n
}

func NewTieDb(dir string, maxDay int) *TieDb {
	mydb, err := db.OpenDB(dir)
	if err != nil {
		panic(err)
	}
	tdb := &TieDb{
		tdb:           mydb,
		RequestTable:  newTable(mydb, "req"),
		ResponseTable: newTable(mydb, "res"),
	}
	if maxDay > 0 {
		maxTime := time.Now().Unix() - int64(maxDay)*86400
		if tdb.RequestTable.Gc(maxTime) > 100 {
			tdb.RequestTable.Scrub()
		}
		if tdb.ResponseTable.Gc(maxTime) > 100 {
			tdb.ResponseTable.Scrub()
		}
	}
	return tdb
}

func (t *TieDb) Flush() {
}

func (t *TieDb) Clean(max_time_unix int64) {
	t.RequestTable.Gc(max_time_unix)
	t.ResponseTable.Gc(max_time_unix)
}

func (t *TieDb) StartGcTimer(sec int64, max_life int64) {
	utils.SetInterval(func() {
		t.Clean(time.Now().Unix() - max_life)
	}, sec)
}

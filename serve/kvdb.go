package serve

import (
	"github.com/HouzuoGuo/tiedot/db"
	"github.com/HouzuoGuo/tiedot/uid"
	"log"
)

type TieDb struct {
	tdb           *db.DB
	RequestTable  *db.Col
	ResponseTable *db.Col
}

func NewTieDb(dir string) *TieDb {
	mydb, err := db.OpenDB(dir)
	if err != nil {
		panic(err)
	}
	if err := mydb.Create("req", 1); err != nil {
		log.Println(err)
	}
	if err := mydb.Create("res", 1); err != nil {
		log.Println(err)
	}
	mydb.Scrub("req")
	mydb.Scrub("res")
	req := mydb.Use("req")
	res := mydb.Use("res")
	tdb := &TieDb{RequestTable: req, ResponseTable: res, tdb: mydb}
	return tdb
}

func (t *TieDb) Flush() {
	t.tdb.Flush()
}

func NextUid() uint64 {
	return uid.NextUID()
}

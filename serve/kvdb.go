package serve

import (
	"github.com/HouzuoGuo/tiedot/db"
	"github.com/HouzuoGuo/tiedot/uid"
	"log"
	  "github.com/hidu/goutils"
	  "time"
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

func (t *TieDb)Clean(max_time_unix int64){
   t.RequestTable.ForAll(func(id uint64, doc map[string]interface{}) bool{
       if(doc["now"].(int64)<max_time_unix){
         t.RequestTable.Delete(id)
         log.Println("delete expire req,",id)
         }
       return true;
   })
   t.ResponseTable.ForAll(func(id uint64, doc map[string]interface{}) bool{
       if(doc["now"].(int64)<max_time_unix){
         t.RequestTable.Delete(id)
         log.Println("delete expire res,",id)
         }
       return true;
   })
}

func (t *TieDb)StartGcTimer(sec int64,max_life int64){
  goutils.SetInterval(func(){
     t.Clean(time.Now().Unix()-max_life)
  },sec)
}

func NextUid() uint64 {
	return uid.NextUID()
}

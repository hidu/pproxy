package serve

import (
	"time"

	"github.com/boltdb/bolt"
	"github.com/hidu/goutils/time_util"
)

type KV_TBALE_NAME_TYPE string

const (
	KV_TABLE_REQ KV_TBALE_NAME_TYPE = "req"
	KV_TABLE_RES KV_TBALE_NAME_TYPE = "res"
)

type kvStore struct {
	dbPath string
	db     *bolt.DB
	tables map[KV_TBALE_NAME_TYPE]*kvStoreTable
}

type kvStoreTable struct {
	name KV_TBALE_NAME_TYPE
	kv   *kvStore
}

type StoreType struct {
	Now  int64  `json:"now"`
	Data KvType `json:"data"`
}

func newStoreType(data map[string]any) *StoreType {
	return &StoreType{Now: time.Now().Unix(), Data: data}
}

func newKvStore(dbPath string) (kv *kvStore, err error) {
	kv = &kvStore{
		dbPath: dbPath,
	}
	kv.db, err = bolt.Open(kv.dbPath, 0600, nil)
	if err != nil {
		return
	}
	kv.tables = make(map[KV_TBALE_NAME_TYPE]*kvStoreTable)

	kv.initTable(KV_TABLE_REQ)
	kv.initTable(KV_TABLE_RES)

	return
}

func (kv *kvStore) initTable(name KV_TBALE_NAME_TYPE) {
	kv.tables[name] = newkvStoreTable(name, kv)
}

func (kv *kvStore) GetkvStoreTable(name KV_TBALE_NAME_TYPE) (tb *kvStoreTable) {
	if tb, has := kv.tables[name]; has {
		return tb
	}
	return nil
}

func (kv *kvStore) Gc(max_life int64) {
	for _, tb := range kv.tables {
		tb.Gc(max_life)
	}
}

func (kv *kvStore) StartGcTimer(sec int64, max_life int64) {
	if max_life < 1 {
		return
	}
	time_util.SetInterval(func() {
		kv.Gc(max_life)
	}, sec)
}

func newkvStoreTable(name KV_TBALE_NAME_TYPE, kv *kvStore) *kvStoreTable {
	tb := &kvStoreTable{
		name: name,
		kv:   kv,
	}
	tb.kv.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(name))
		return err
	})
	return tb
}

func (tb *kvStoreTable) Save(key []byte, val *StoreType) error {
	err := tb.kv.db.Update(func(tx *bolt.Tx) error {
		bk, _ := tx.CreateBucketIfNotExists([]byte(tb.name))
		return bk.Put(key, dataEncode(val))
	})
	return err
}

func (tb *kvStoreTable) Get(key []byte) (val *StoreType, err error) {
	err = tb.kv.db.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(tb.name))
		bs := bk.Get(key)
		if len(bs) > 0 {
			return dataDecode(bs, &val)
		}
		return nil
	})
	return
}

func (tb *kvStoreTable) Del(key []byte) (err error) {
	err = tb.kv.db.Update(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(tb.name))
		return bk.Delete(key)
	})
	return
}

func (tb *kvStoreTable) Gc(gc_life int64) {
	if gc_life < 1 {
		return
	}
	max_time := time.Now().Unix() - gc_life
	var val *StoreType
	tb.kv.db.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(tb.name))
		bk.ForEach(func(k, v []byte) error {
			dataDecode(v, &val)
			if val != nil && val.Now < max_time {
				tb.Del(k)
			}
			return nil
		})
		return nil
	})
}

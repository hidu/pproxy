package cache

import (
	"crypto/md5"
	"encoding/hex"
	//   "fmt"
	"bytes"
	"encoding/gob"
	"github.com/hidu/goutils"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type FileCache struct {
	data_dir    string
	gc_interval int64
	Cache
	mu sync.Mutex
}

func NewFileCache(data_dir string) *FileCache {
	cache := &FileCache{data_dir: data_dir, gc_interval: 3600}
	return cache
}
func (cache *FileCache) Set(key string, data []byte, life int64) (suc bool) {
	//    log.Println("cache set ",key,data)
	defer cache.mu.Unlock()
	cache_path := cache.genCachePath(key)
	cache.mu.Lock()
	f, err := os.OpenFile(cache_path, os.O_CREATE|os.O_RDWR, 0644)
	defer f.Close()
	if err != nil {
		p_dir := path.Dir(cache_path)
		os.MkdirAll(p_dir, 0755)
		f, err = os.OpenFile(cache_path, os.O_CREATE|os.O_RDWR, 0644)
		defer f.Close()
	}
	var bf bytes.Buffer
	enc := gob.NewEncoder(&bf)
	now := time.Now().Unix()
	cdata := Data{key, data, now, life}
	enc.Encode(cdata)
	f.Write(bf.Bytes())
	return true
}

func (cache *FileCache) Get(key string) (has bool, data []byte) {
	//    log.Println("cache get ",key)
	cache_path := cache.genCachePath(key)
	return cache.getByPath(cache_path)
}

func (cache *FileCache) Delete(key string) bool {
	defer cache.mu.Unlock()
	cache_path := cache.genCachePath(key)
	cache.mu.Lock()
	_, err := os.Stat(cache_path)
	if err != nil {
		log.Println("delete cache err:", err)
		return true
	}
	e1 := os.Remove(cache_path)
	return e1 == nil
}

func (cache *FileCache) DeleteAll() bool {
	err := os.RemoveAll(cache.data_dir)
	if err != nil {
		log.Println("delete all file cache err:", err)
	}
	return true
}

func (cache *FileCache) genCachePath(key string) string {
	h := md5.New()
	h.Write([]byte(key))
	md5_str := hex.EncodeToString(h.Sum(nil))
	file_path := cache.data_dir + "/" + string(md5_str[:3]) + "/" + md5_str
	return file_path
}

func (cache *FileCache) getByPath(file_path string) (has bool, data []byte) {
	defer cache.mu.Unlock()
	cache.mu.Lock()
	f, err := os.Open(file_path)
	defer f.Close()
	if err != nil {
		return
	}
	data_bf, err1 := ioutil.ReadAll(f)
	if err1 != nil {
		log.Println("read cache file failed:", file_path, err1.Error())
		return
	}
	dec := gob.NewDecoder(bytes.NewBuffer(data_bf))
	var cache_data Data
	err = dec.Decode(&cache_data)
	if err != nil {
		return
	}
	if time.Now().Unix()-cache_data.Life > cache_data.CreateTime {
		return false, cache_data.Data
	}
	return true, cache_data.Data
}

func (cache *FileCache) GC() {
	info, err := os.Stat(cache.data_dir)
	if err != nil || !info.IsDir() {
		return
	}
	filepath.Walk(cache.data_dir, func(file_path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			has, data := cache.getByPath(file_path)
			if has || len(data) > 0 {
				os.Remove(file_path)
			}

		}
		return nil
	})
}

func (cache *FileCache) StartGcTimer(sec int64) {
	utils.SetInterval(cache.GC, sec)
}

package cache

type Cache interface {
	Set(key string, val []byte, life int64) (suc bool)
	Get(key string) (has bool, data []byte)
	Delete(key string) (suc bool)
	DeleteAll() (suc bool)
	GC()
	StartGcTimer(sec int64)
}

type Data struct {
	Key        string
	Data       []byte
	CreateTime int64
	Life       int64
}

var defaultCache Cache = new(NoneCache)

func SetDefaultCacheHandler(cache Cache) {
	defaultCache = cache
}

func Set(key string, val []byte, life int64) (suc bool) {
	return defaultCache.Set(key, val, life)
}

func Get(key string) (has bool, data []byte) {
	return defaultCache.Get(key)
}

func Delete(key string) (suc bool) {
	return defaultCache.Delete(key)
}
func GC() {
	defaultCache.GC()
}

package cache

type NoneCache struct {
	Cache
}

func NewNoneCache() *NoneCache {
	return &NoneCache{}
}
func (cache *NoneCache) Set(key string, val []byte, life int64) bool {
	return true
}

func (cache *NoneCache) Get(key string) (has bool, data []byte) {
	return false, nil
}
func (cache *NoneCache) Delete(key string) (suc bool) {
	return true
}

func (cache *NoneCache) DeleteAll() (suc bool) {
	return true
}

func (cache *NoneCache) GC() {

}
func (cache *NoneCache) StartGcTimer(sec int64) {
}

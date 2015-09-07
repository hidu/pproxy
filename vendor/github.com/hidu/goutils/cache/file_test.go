package cache

import (
	"os"
	"testing"
	//    "fmt"
)

func Test_fileCache(t *testing.T) {
	cc := NewFileCache(os.TempDir() + "/goutils_cache/")
	cc.Set("a", []byte("aaa"), 100)
	Set("bbb", []byte("bbbb"), 100)
	SetDefaultCacheHandler(cc)
	Set("ccc", []byte("ccc"), 100)
	cc.DeleteAll()

}

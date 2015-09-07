package utils

import (
	//      "fmt"
	"encoding/json"
	"github.com/bmizerany/assert"
	"testing"
)

func TestGetVal(t *testing.T) {
	str := `{"a":{"c":1,"f":1.1},"b":[1,2],"d":{"1":{"a":"ccc"},"2":3},"e":[]}`
	var m map[string]interface{}
	err := json.Unmarshal([]byte(str), &m)
	assert.Equal(t, err, nil)

	var val interface{}
	has := false

	w := NewInterfaceWalker(m)
	cases := make(map[string]interface{})
	cases["a/c"] = "1"
	cases["b/1"] = "2"
	cases["b/0"] = "1"
	cases["d//1/a"] = "ccc"
	for k, v := range cases {
		for _, _to := range []string{"", "/", "//", "/.", "/./", "../..", "..", "../"} {
			w.GoInto(_to)
			for _, _suffix := range []string{"", "/", "//", "/.", "/./"} {
				val, has = w.GetString(k + _suffix)
				assert.Equal(t, v, val)
				assert.Equal(t, true, has)

				val, has = w.GetString(k + _suffix + "/unkonwn")
				assert.Equal(t, "", val)
				assert.Equal(t, false, has)
			}
		}
	}
	//////////////////////////////////////////////////
	cases1 := make(map[string]interface{})
	cases1["b"] = []int{1, 2}
	cases1["e"] = []int{}
	for k, v := range cases1 {
		for _, _suffix := range []string{"", "/", "//", "/.", "/./"} {
			val, has = w.GetIntSlice(k + _suffix)
			assert.Equal(t, v, val)
			assert.Equal(t, true, has)
		}
	}

	int_map := make(map[int]int)
	int_map[1] = 2
	int_map[2] = 9
	slice_walker := NewInterfaceWalker(int_map)
	for k, v := range int_map {
		val, has = slice_walker.GetInt(k)
		assert.Equal(t, v, val)
		assert.Equal(t, true, has)
	}

	arr := []int{1, 3, 5}
	walker_3 := NewInterfaceWalker(arr)
	for i, v := range arr {
		val, has = walker_3.GetInt(i)
		assert.Equal(t, v, val)
		assert.Equal(t, true, has)
	}
	val, has = walker_3.GetIntSlice("")
	assert.Equal(t, arr, val)
	assert.Equal(t, true, has)

	w.GoInto("a")
	val, has = w.GetString("c")
	assert.Equal(t, "1", val)
	assert.Equal(t, true, has)
	val, has = w.GetInt("c")
	assert.Equal(t, 1, val)
	assert.Equal(t, true, has)
}

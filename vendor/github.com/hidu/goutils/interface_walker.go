package utils

/**
*toolkit for the interface{}
 */
import (
	"fmt"
	"log"
	os_path "path"
	"reflect"
	"strconv"
	"strings"
)

type Object struct {
	data  interface{}
	paths []string
}

func NewInterfaceWalker(obj interface{}) *Object {
	return &Object{data: obj, paths: []string{"/"}}
}

func (obj *Object) GetObject(path interface{}) (val *Object, has bool) {
	tmp, has := obj.GetInterface(path)
	if has {
		val = NewInterfaceWalker(tmp)
	}
	return
}

func (obj *Object) GoInto(path interface{}) {
	path_orign := fmt.Sprint(path)
	path_str := path_orign
	if len(path_orign) == 0 || path_orign[0] != '/' {
		path_str = strings.Join(obj.paths, "/") + "/" + path_orign
	}
	obj.paths = strings.Split(os_path.Clean(path_str), "/")
}

func (obj *Object) GetInterface(path interface{}) (val interface{}, has bool) {
	path_str := fmt.Sprint(path)
	if len(path_str) > 0 && path_str[0] != '/' {
		path_str = strings.Join(obj.paths, "/") + "/" + path_str
	}
	val, has = InterfaceWalk(obj.data, path_str)
	return
}

func (obj *Object) GetInterfaceSlice(path interface{}, def ...[]interface{}) (val []interface{}, has bool) {
	tmp, has := obj.GetInterface(path)
	if has {
		_type := reflect.TypeOf(tmp).String()
		if len(_type) < 3 || _type[:2] != "[]" {
			log.Printf("GetStringArray failed,[%s] not slice", _type)
		} else {
			_value := reflect.ValueOf(tmp)
			val_len := _value.Len()
			result := make([]interface{}, val_len)
			for i := 0; i < val_len; i++ {
				result[i] = _value.Index(i).Interface()
			}
			return result, true
		}
	}
	if len(def) > 0 {
		return def[0], false
	} else {
		return []interface{}{}, false
	}
}

/*
*读取指定项的值
 */
func (obj *Object) GetString(path interface{}, def ...string) (val string, has bool) {
	tmp, has := obj.GetInterface(path)
	if has {
		return fmt.Sprint(tmp), true
	}
	if len(def) > 0 {
		return def[0], false
	} else {
		return "", false
	}
}

func (obj *Object) GetStringSlice(path interface{}, def ...[]string) (val []string, has bool) {
	interface_array, has := obj.GetInterfaceSlice(path)
	if has {
		result := make([]string, len(interface_array))
		for i, v := range interface_array {
			result[i] = fmt.Sprint(v)
		}
		return result, true
	} else {
		if len(def) > 0 {
			return def[0], false
		} else {
			return []string{}, false
		}
	}
}

func (obj *Object) GetInt(path interface{}, def ...int) (val int, has bool) {
	tmp, has := obj.GetFloat(path)
	if has {
		return int(tmp), true
	} else {
		if len(def) > 0 {
			return def[0], false
		} else {
			return -1, false
		}
	}
}

func (obj *Object) GetIntSlice(path interface{}, def ...[]int) (val []int, has bool) {
	float_arr, has := obj.GetFloatSlice(path)
	if has {
		result := make([]int, len(float_arr))
		for i, v := range float_arr {
			result[i] = int(v)
		}
		return result, true
	} else {
		if len(def) > 0 {
			return def[0], false
		} else {
			return []int{}, false
		}
	}
}

func (obj *Object) GetFloat(path interface{}, def ...float64) (val float64, has bool) {
	str, has := obj.GetString(path)
	if has {
		ret, err := strconv.ParseFloat(str, 64)
		if err != nil {
			log.Printf("GetFloat faild [%s] value:[%v]", path, ret)
			return -1, false
		}
		return ret, true
	} else {
		if len(def) > 0 {
			return def[0], false
		} else {
			return -1, false
		}
	}
}

func (obj *Object) GetFloatSlice(path interface{}, def ...[]float64) (val []float64, has bool) {
	str_arr, has := obj.GetStringSlice(path)
	if has {
		result := make([]float64, len(str_arr))
		for i, v := range str_arr {
			ret, _ := strconv.ParseFloat(v, 64)
			result[i] = ret
		}
		return result, true
	} else {
		if len(def) > 0 {
			return def[0], false
		} else {
			return []float64{}, false
		}
	}
}

func (obj *Object) GetBool(path interface{}) bool {
	val, has := obj.GetString(path)
	if !has {
		return false
	}
	bv, err := strconv.ParseBool(val)
	if err == nil {
		return bv
	}
	return false
}

/**
*quick get the val from a interface
 */
func InterfaceWalk(obj interface{}, path interface{}) (val interface{}, has bool) {
	path_orign := fmt.Sprint(path)
	path_str := strings.TrimSpace(os_path.Clean(path_orign))
	//    fmt.Println("path:",path_str)
	if path_str == "/" || path_str == "." {
		return obj, true
	}
	val_tmp := obj
	paths := strings.Split(strings.Trim(path_str, "/"), "/")
	n := 0
	for i, sub_name := range paths {
		_type := reflect.TypeOf(val_tmp).String()
		_value := reflect.ValueOf(val_tmp)
		if len(_type) > 3 && _type[:3] == "map" {
			has_match := false
			for _, _key := range _value.MapKeys() {
				_key_str := fmt.Sprint(_key.Interface())
				if sub_name == _key_str {
					val_tmp = _value.MapIndex(_key).Interface()
					has_match = true
					break
				}
			}
			if !has_match {
				break
			}
		} else if len(_type) > 3 && _type[:2] == "[]" {
			index, err := strconv.Atoi(sub_name)
			if err != nil {
				log.Printf("now here is slice,[%s] must int,input path is:[%s]", sub_name, path)
				break
			}
			total_len := _value.Len()
			if (index > 0 && index > total_len) || (index < 0 && index*-1 > total_len) {
				log.Printf("slice index out of range,index:[%d],slice size:[%d]", index, total_len)
				break
			}
			if index < 0 {
				index = total_len + index
			}
			val_tmp = _value.Index(index).Interface()
		} else {
			log.Printf("date type error,not map or slice,[%s]=(%v,type=%T),find=%s", strings.Join(paths[:i], "/"), val_tmp, val_tmp, sub_name)
			break
		}
		n++
	}
	if n == len(paths) {
		return val_tmp, true
	}
	return nil, false
}

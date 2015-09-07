package jsonutils

import (
	"testing"
	"encoding/json"
//	"fmt"
	"github.com/bmizerany/assert"
	"github.com/xeipuuv/gojsonschema"
)

func TestGenJsonSchema(t *testing.T){
	str:=`
		{
		"k1":"a",
		"k2":1,
		"k3":[],
		"k4":["a"],
		"k5":[1],
		"k6":[1.1],
		"k7":{},
		"k8":{"a":1},
		"k9":{"a":1,"b":[]},
		"k10":{"a":1,"b":[],"c":{"d":1.1}},
		"k11":{"a":1,"b":[],"c":{"d":1.1,"f":["a"]}},
		"k12":{"a":1,"b":[{"a":1,"b":[1]}]}
		}
	`
	var obj interface{}
	err:=json.Unmarshal([]byte(str),&obj)
	assert.Equal(t, err, nil)
	
	schema,err:=GenJsonSchema(obj)
	assert.Equal(t, err, nil)
	
	_,err=json.MarshalIndent(schema,"","  ")
	assert.Equal(t, err, nil)
//	fmt.Println(string(bs))
	
	goSchema,err:=gojsonschema.NewSchema(gojsonschema.NewGoLoader(schema))
	assert.Equal(t, err, nil)
	documentLoader:=gojsonschema.NewStringLoader(str)
	ret,err:=goSchema.Validate(documentLoader)
	assert.Equal(t, err, nil)
	assert.Equal(t, ret.Valid(), true)
}

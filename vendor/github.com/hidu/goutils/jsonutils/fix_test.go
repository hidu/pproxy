package jsonutils

import (
	"testing"
	"encoding/json"
	"fmt"
	"github.com/bmizerany/assert"
)


type DataStrcut struct {
        A   int64    `json:"a"`
        B   []string `json:"b"`
        Fff []string `json:"fff"`
        G   struct {
                A1 int64 `json:"a1"`
                A2 int64 `json:"a2"`
        } `json:"g"`
        Gg []struct {
                D int64 `json:"d"`
        } `json:"gg"`
}

func TestFixDataWithSchema(t *testing.T){
	dataStr:=`{
    "a": 145,
    "b": [
        "d",
        "e"
    ],
    "fff": "hello",
    "gg": {"d":789},
    "g":{
    	"a1":123,
    	"a2":456
    }
}`
	schemaStr:=`
	{
    "properties": {
        "a": {
            "type": "string"
        },
        "b": {
            "items": {
                "type": "string"
            },
            "type": "array"
        },
        "fff": {
            "items": {
                "type": "string"
            },
            "type": "array"
        },
        "gg": {
            "type": "array",
            "items": {
                "type": "object",
                "properties":{
                	"d":{
                		"type":"integer"
                	}
                }
            }
        }
    },
    "type": "object"
}
	`
	var data interface{}
	err:=json.Unmarshal([]byte(dataStr),&data)
	assert.Equal(t, err, nil)
//	fmt.Println("data:",data)
	
	var schema interface{}
	err=json.Unmarshal([]byte(schemaStr),&schema)
	assert.Equal(t, err, nil)
//	fmt.Println("schema:",schema)
	
	dataNew,err:=FixDataWithSchema(data,schema)
	assert.Equal(t, err, nil)
	
	dbs,err:=json.Marshal(dataNew)
	assert.Equal(t, err, nil)
	
	fmt.Println(string(dbs))
	
	var myDs DataStrcut
	err=json.Unmarshal(dbs,&myDs)
	assert.Equal(t, err, nil)
	
	assert.Equal(t,145,int(myDs.A))
	
	assert.Equal(t,1,len(myDs.Gg))
	
	assert.Equal(t,789,int(myDs.Gg[0].D))

	assert.Equal(t,123,int(myDs.G.A1))
	assert.Equal(t,456,int(myDs.G.A2))
	
}

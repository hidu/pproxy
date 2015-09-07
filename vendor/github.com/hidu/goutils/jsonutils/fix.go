/**
*fix json data with jsonschema
*/
package jsonutils

import (
	"reflect"
	"fmt"
	"github.com/xeipuuv/gojsonschema"
)


func FixDataWithSchema(data interface{},schema interface{})(dataFix interface{},err error){
	_,err=gojsonschema.NewSchema(gojsonschema.NewGoLoader(schema))
	if(err!=nil){
		return nil,err
	}
	
	return fixDataWithSchemaInterface(data,schema)
}

func fixDataWithSchemaInterface(data interface{},schema interface{})(dataFix interface{},err error){
	t:=reflect.TypeOf(schema).Kind()
	if(t!=reflect.Map){
		return nil,fmt.Errorf("type error,%s",t)
	}
	schemaMap:=schema.(map[string]interface{})
	
	schemaDataType,hasKey:=schemaMap["type"]
	if(!hasKey){
		return nil,fmt.Errorf("schema miss 'type' prop")
	}
	
	dataKind:=reflect.TypeOf(data).Kind()
	
	if(schemaDataType=="object"){
		dataNew:=make(map[string]interface{})
		properties,hasKey:=schemaMap["properties"]
		if(!hasKey){
			return nil,fmt.Errorf("miss properties")
		}
		if(dataKind==reflect.Map){
			dataMap:=data.(map[string]interface{})
			propertiesMap:=properties.(map[string]interface{})
			for dataK,dataV:=range dataMap{
				itemSchema,hasKey:=propertiesMap[dataK]
				if(hasKey){
					dataNew[dataK],err=fixDataWithSchemaInterface(dataV,itemSchema)
				}else{
					dataNew[dataK]=dataV
				}
			}
		}else{
			return nil,fmt.Errorf("unknow schemaDataType:",schemaDataType,"data:",data,"schema:",schema)
		}
		dataFix=dataNew
		
	}else if(schemaDataType=="array"){
		items,hasKey:=schemaMap["items"]
		if(!hasKey){
			return nil,fmt.Errorf("miss items")
		}
		if(dataKind!=reflect.Slice){
			d:=make([]interface{},0)
			d=append(d,data)
			data=d
		}
		dataNew:=make([]interface{},0)
		for _,dataItem:=range data.([]interface{}){
			itemNew,err:=fixDataWithSchemaInterface(dataItem,items)
			if(err!=nil){
				return nil,err
			}
			dataNew=append(dataNew,itemNew)
		}
		dataFix=dataNew
	}else{
		dataFix=data
	}
	return
}

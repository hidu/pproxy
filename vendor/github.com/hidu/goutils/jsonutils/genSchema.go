package jsonutils

import(
	"reflect"
	"fmt"
)

func GenJsonSchema(data interface{})(schema interface{},err error){
 	kind:=reflect.TypeOf(data).Kind()
 	sc:=make(map[string]interface{})
 	sc["description"]=""
 	switch(kind){
 		case reflect.Map:
 			sc["type"]="object"
 			properties:=make(map[string]interface{})
 			dataMap:=data.(map[string]interface{})
 			for dataK,dataV:=range dataMap{
 				properties[dataK],err=GenJsonSchema(dataV)
 				if(err!=nil){
 					return
 				}
 			}
 			sc["properties"]=properties
 			sc["required"]=make([]string,0)
 		case reflect.Slice:
 			sc["type"]="array"
 			sc["minItems"]=0
 			
 			var items interface{}
 			dataSlice:=data.([]interface{})
 			
 			if(len(dataSlice)>0){
 				items,err=GenJsonSchema(dataSlice[0])
 				if(err!=nil){
 					return
 				}
 			}else{
 				items=make(map[string]interface{})
 			}
 			
 			sc["items"]=items
 		case reflect.String:
 			 sc["type"]="string"
 			 schema=sc
 		case reflect.Int:
 		case reflect.Int16:
 		case reflect.Int32:
 		case reflect.Int64:
 		case reflect.Int8:
 		case reflect.Uint:
 		case reflect.Uint16:
 		case reflect.Uint32:
 		case reflect.Uint64:
 		case reflect.Uint8:
 			 sc["type"]="integer"
 			 sc["minimum"]=0
 		case reflect.Bool:
 			 sc["type"]="boolean"
 			 schema=sc
 		case reflect.Float32:	 
 		case reflect.Float64:	 
 			 sc["type"]="number"
 			 sc["minimum"]=0
 		default:
 			err=fmt.Errorf("unsupport type %s",kind.String())
 	}
 	
 	schema=sc
 	
 	return 
}

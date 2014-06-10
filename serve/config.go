package serve

import (
    "encoding/json"
    "github.com/hidu/goutils"
    "log"
)

type Config struct{
  Port      int     `json:"port"`
  Title     string  `json:"title"`
  Notice    string  `json:"notice"`
  AuthType  int     `json:"authType"`
  DataDir   string     `json:"dataDir"`
  ResponseSave  int     `json:"responseSave"` 
}
const (
   AuthType_NO=0
   AuthType_Basic=1
   AuthType_BasicWithAny=2
   
   ResponseSave_All=0
   ResponseSave_HasBroad=1  //has show
)
//"0:no auth | 1:basic auth | 2:basic auth with any name"

func LoadConfig(confPath string) (*Config, error){
    data,err:=goutils.File_get_contents(confPath)
    if(err!=nil){
       log.Println("load config",confPath,"failed,err:",err)
       return nil,err
    }
    var config *Config
    err=json.Unmarshal(data,&config)
    if(err!=nil){
        log.Println("config is  incorrect",err)
        return nil,err
    }
    return config,nil
}
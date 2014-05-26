package main
import (
 "github.com/hidu/pproxy/serve"
 "flag"
)
var port=flag.Int("port",8080,"main proxy port")
var data_dir=flag.String("data","./data/","the datato save")
var filterPath=flag.String("js","./rewrite.js","you can change the req with js code")
var store_sec=flag.Int64("store_sec",7200,"req and res store time")
func main(){
  flag.Parse()
  ser:=serve.NewProxyServe(*data_dir,*filterPath,*port)
  ser.Start()
}
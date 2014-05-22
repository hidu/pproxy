package main
import (
 "github.com/hidu/pproxy/serve"
 "flag"
)
var port=flag.Int("port",8080,"main proxy port")
var filterPath=flag.String("js","rewrite.js","filter js path")
var store_sec=flag.Int64("store_sec",7200,"req and res store time")
func main(){
  flag.Parse()
  ser:=serve.NewProxyServe(*filterPath,*store_sec)
  ser.Port=*port
  ser.Start()
}
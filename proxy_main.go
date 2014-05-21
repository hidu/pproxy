package main
import (
 "github.com/hidu/pproxy/serve"
 "flag"
)
var port=flag.Int("port",8080,"main proxy port")
var filterPath=flag.String("js","","filter js path")
func main(){
  flag.Parse()
  ser:=serve.NewProxyServe(*filterPath)
  ser.Port=*port
  ser.Start()
}
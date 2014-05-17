package main
import (
 "github.com/hidu/pproxy/serve"
 "flag"
)
var port=flag.Int("port",8080,"main proxy port")
func main(){
  flag.Parse()
  ser:=serve.NewProxyServe()
  ser.Port=*port
  ser.Start()
}
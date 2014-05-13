package main
import (
 "github.com/hidu/sproxy/proxyserve"
)
func main(){
  ser:=proxyserve.NewProxySer()
  ser.Port=8080
  ser.Start()
}
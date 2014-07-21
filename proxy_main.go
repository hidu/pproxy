package main

import (
	"./serve"
	"flag"
	"fmt"
	"log"
	"os"
)

var configPath = flag.String("conf", "./conf/pproxy.ini", "json config path")
var port = flag.Int("port", 0, "proxy port")
var debug = flag.Bool("debug", false, "debug the request")
var show_conf=flag.Bool("demo_conf",false,"show default conf")

var version=flag.Bool("v",false,"show version")

func main() {
	flag.Parse()
	
	if(*show_conf){
	  demo_conf:=serve.GetDemoConf()
	  fmt.Println("##########################################################")
	  fmt.Println("                  pproxy demo conf")
	  fmt.Println("##########################################################")
	  fmt.Println(demo_conf)
	  os.Exit(0)
	}
	
	if(*version){
	   fmt.Println("pproxy version:",serve.GetVersion())
	   os.Exit(0)
	}
	
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Ldate)
	ser, err := serve.NewProxyServe(*configPath, *port)
	if err != nil {
		fmt.Println("start pproxy failed", err)
		os.Exit(2)
	}
	ser.Debug = *debug
	ser.Start()
}

package main

import (
	"./serve"
	"flag"
	"fmt"
	"os"
)

var configPath = flag.String("conf", "./conf/config.json", "json config path")
var port = flag.Int("port", 0, "main proxy port")

var debug = flag.Bool("debug", false, "debug the request")

func main() {
	flag.Parse()
	ser, err := serve.NewProxyServe(*configPath, *port)
	if err != nil {
		fmt.Println("start pproxy failed", err)
		os.Exit(2)
	}
	ser.Debug = *debug
	ser.Start()
}

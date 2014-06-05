package main

import (
    "flag"
    "./serve"
)

var port = flag.Int("port", 8080, "main proxy port")
var data_dir = flag.String("data", "./data/", "the data to save")
var filterPath = flag.String("js", "./rewrite.js", "you can change the req with js code")
var store_sec = flag.Int64("store_sec", 7200, "req and res store time")
var authType = flag.Int("auth", 0, "0:no auth | 1:basic auth | 2:basic auth with any name")
var debug = flag.Bool("debug", false, "debug the request")

func main() {
    flag.Parse()
    ser := serve.NewProxyServe(*data_dir, *filterPath, *port)
    ser.AuthType = *authType
    ser.Debug = *debug
    ser.Start()
}

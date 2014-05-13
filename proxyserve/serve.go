package proxyserve
import (
	"github.com/elazarl/goproxy"
//	"github.com/elazarl/goproxy/ext/auth"
	"fmt"
	"net/http"
	"log"
)

type ProxyServe struct{
   Port int
   Goproxy *goproxy.ProxyHttpServer
}

func (ser *ProxyServe)Start(){
	ser.Goproxy = goproxy.NewProxyHttpServer()
	ser.Goproxy.OnRequest().DoFunc(func(r *http.Request,ctx *goproxy.ProxyCtx) (*http.Request, *http.Response){
		author_info:=getAuthorInfo(r)
		for k,_:=range r.Header{
		   if(len(k)>5 && k[:6]=="Proxy-"){
		      r.Header.Del(k)
		   }
		}
//		if(author_info==nil){
//			return nil,auth.BasicUnauthorized(r,"auth need")
//		}
		return r,nil
	})
	
	ser.Goproxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
//		fmt.Println(resp.Request.Method, resp.Request.URL,resp.ContentLength)
		return resp
	})
	
	addr:=fmt.Sprintf("%s:%d","",ser.Port)
	log.Println("proxy listen at ",addr)
	err:=http.ListenAndServe(addr, ser.Goproxy)
	log.Println(err)
}

func NewProxySer()*ProxyServe{
  return new(ProxyServe)
}
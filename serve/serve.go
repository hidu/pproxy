package serve
import (
	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
	"github.com/HouzuoGuo/tiedot/db"
	"fmt"
	"net/http"
//	"net/http/httputil"
	"log"
	"time"
//	"encoding/gob"
//	"bytes"
	"net"
	"strconv"
//	"io/ioutil"
	"reflect"
)

type ProxyServe struct{
   Port int
   Goproxy *goproxy.ProxyHttpServer
   AdminName string
   AdminPsw string
   mydb *TieDb
}

type TieDb struct{
    RequestTable *db.Col
    ResponseTable *db.Col
}
type kvType map[string]interface{}

func (ser *ProxyServe) ServeHTTP(w http.ResponseWriter, req *http.Request) {
      host,port,_:=net.SplitHostPort(req.Host)
		port_int,_:=strconv.Atoi(port)
		isLocalReq:=port_int==ser.Port
		if(isLocalReq){
		  isLocalReq=IsLocalIp(host)
		}
		if(isLocalReq){
		    handleLocalReq(w,req)
		}else{
		 ser.Goproxy.ServeHTTP(w,req)
	}
}

func (ser *ProxyServe)Start(){
	ser.Goproxy = goproxy.NewProxyHttpServer()
	ser.Goproxy.OnRequest().DoFunc(func(r *http.Request,ctx *goproxy.ProxyCtx) (*http.Request, *http.Response){
		authInfo:=getAuthorInfo(r)
		ctx.UserData="guest"
		if(authInfo!=nil){
			ctx.UserData=authInfo.Name
		}
		for k,_:=range r.Header{
		   if(len(k)>5 && k[:6]=="Proxy-"){
		      r.Header.Del(k)
		   }
		}
		if(ser.AdminName!="" && (authInfo==nil|| (authInfo!=nil && !authInfo.isEqual(ser.AdminName,ser.AdminPsw)))){
			return nil,auth.BasicUnauthorized(r,"auth need")
		}
		
		ser.logRequest(r,ctx)
		return r,nil
	})
	
	ser.Goproxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if(resp==nil || resp.Request==nil){
		  return resp
		}
      ser.logResponse(resp,ctx)
		return resp
	})
	
	addr:=fmt.Sprintf("%s:%d","",ser.Port)
	log.Println("proxy listen at ",addr)
	
	err:=http.ListenAndServe(addr,ser)
	log.Println(err)
}


func (ser *ProxyServe)logRequest(req *http.Request,ctx *goproxy.ProxyCtx){
  log.Println(ctx.Session,req.URL.String())
   data:=kvType{}
   data["session_id"]=ctx.Session
   data["req_start"]=time.Now().UnixNano()
   data["host"]=req.Host
   data["header"]=req.Header
   data["url"]=req.URL.String()
   data["cookie"]=req.Cookies()
   data["user"]=ctx.UserData.(string)
  id,err:= ser.mydb.RequestTable.Insert(data)
  if(err!=nil){
    log.Println(err)
    return
  }
  ctx.UserData=id
}
/**
*log response if the req has log
*/
func (ser *ProxyServe)logResponse(res *http.Response, ctx *goproxy.ProxyCtx){
   if(reflect.TypeOf(ctx.UserData).Kind()!=reflect.Uint64){
     return
   }
   req_doc_id:=ctx.UserData.(uint64)
   data:=kvType{}
   data["req_doc_id"]=req_doc_id
   data["session_id"]=ctx.Session
   data["res_start"]=time.Now().UnixNano()
   data["header"]=res.Header
   id,err:= ser.mydb.ResponseTable.Insert(data)
   if(err!=nil){
	    log.Println(err)
	    return
  }
  var req_data kvType
  ser.mydb.RequestTable.Read(req_doc_id,&req_data)
  if(req_data!=nil){
   req_data["res_doc_id"]=id
  }
  ser.mydb.RequestTable.Update(req_doc_id,req_data)
}

func NewProxyServe()*ProxyServe{
   proxy:= new(ProxyServe)
   proxy.mydb=NewTieDb("./data/")
  return proxy
}

func NewTieDb(dir string) *TieDb{
   mydb, err := db.OpenDB(dir)
	if err != nil {
		panic(err)
	}
	if err :=mydb.Create("req", 1); err != nil {
	 log.Println(err)
	}
	if err := mydb.Create("res", 1); err != nil {
		log.Println(err)
	}
	mydb.Scrub("req")
	mydb.Scrub("res")
	req := mydb.Use("req")
	res := mydb.Use("res")
	tdb:=&TieDb{RequestTable:req,ResponseTable:res}
	return tdb
}
package serve
import (
	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
	"fmt"
	"net/http"
	"log"
	"time"
	"net"
	"strconv"
	"reflect"
	"github.com/googollee/go-socket.io"
)

type ProxyServe struct{
   Port int
   Goproxy *goproxy.ProxyHttpServer
   AdminName string
   AdminPsw string
   mydb *TieDb
   ws *socketio.SocketIOServer
   wsClients map[string]*wsClient
}
type wsClient struct{
  ns *socketio.NameSpace
  user string
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
		    ser.handleLocalReq(w,req)
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
	ser.initWs()
	err:=http.ListenAndServe(addr,ser)
	log.Println(err)
}


func (ser *ProxyServe)logRequest(req *http.Request,ctx *goproxy.ProxyCtx){
  
   req_uid:=uint64(time.Now().Unix()+ctx.Session)
   data:=kvType{}
   data["session_id"]=ctx.Session
   data["req_start"]=time.Now().UnixNano()
   data["host"]=req.Host
   data["header"]=req.Header
   data["url"]=req.URL.String()
   data["cookie"]=req.Cookies()
   data["user"]=ctx.UserData.(string)
   err:= ser.mydb.RequestTable.InsertRecovery(req_uid,data)
   log.Println(ctx.Session,req.URL.String(),"req_docid=",req_uid,err)
   if(err!=nil){
     log.Println(err)
     return
   }
  ser.Broadcast_Req(ctx.Session,req,req_uid)
  ctx.UserData=req_uid
}
/**
*log response if the req has log
*/
func (ser *ProxyServe)logResponse(res *http.Response, ctx *goproxy.ProxyCtx){
   if(reflect.TypeOf(ctx.UserData).Kind()!=reflect.Uint64){
     return
   }
   req_uid:=ctx.UserData.(uint64)
   data:=kvType{}
   data["session_id"]=ctx.Session
   data["res_start"]=time.Now().UnixNano()
   data["header"]=res.Header
   
   buf:=forgetRead(&res.Body)
   data["body"]=buf.String()
   err:= ser.mydb.ResponseTable.InsertRecovery(req_uid,data)
   log.Println("save response [",req_uid,"]",err)
   if(err!=nil){
	    log.Println(err)
	    return
  }
  ser.GetResponseByReqDocid(req_uid)
}

func (ser *ProxyServe)GetResponseByReqDocid(docid uint64) (res_data kvType){
  ser.mydb.ResponseTable.Read(docid,&res_data)
 return res_data
}

func NewProxyServe()*ProxyServe{
   proxy:= new(ProxyServe)
   proxy.mydb=NewTieDb("./data/")
  return proxy
}


func (ser *ProxyServe)Broadcast_Req(id int64,req *http.Request,docid uint64){
  data:=make(map[string]interface{})
  data["docid"]=docid
  data["sid"]=id
  data["host"]=req.Host
  data["path"]=req.URL.Path
  data["method"]=req.Method
  for _,client:=range ser.wsClients{
     send_req(client,data)
  }
}
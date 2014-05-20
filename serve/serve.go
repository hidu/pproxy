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
	"strings"
)

type ProxyServe struct{
   Port int
   Goproxy *goproxy.ProxyHttpServer
   AdminName string
   AdminPsw string
   mydb *TieDb
   ws *socketio.SocketIOServer
   wsClients map[string]*wsClient
   startTime time.Time
   
   MaxResSaveLength int64
   
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
		req_new:=ser.changeRequest(r,ctx)
		
		ser.logRequest(r,ctx,req_new)
		return req_new,nil
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

func (ser *ProxyServe)changeRequest(req *http.Request,ctx *goproxy.ProxyCtx )*http.Request{
    if(strings.HasPrefix(req.URL.Path,"/napi")){
	  req.URL.Host="beta.zhidao.baidu.com:80"
	  req.URL.Path="/rds"+req.URL.Path
	}
	if(strings.HasPrefix(req.URL.Path,"/qas")){
	  req.URL.Host="beta.zhidao.baidu.com:80"
	  req.URL.Path="/rds"+req.URL.Path[4:]
	}
	return req
}

func getReqLogData(req *http.Request) kvType{
   req.ParseForm()
   data:=kvType{}
   data["host"]=req.Host
   data["header"]=map[string][]string(req.Header)
   data["url"]=req.URL.String()
   data["cookies"]=req.Cookies()
   data["client_ip"]=req.RemoteAddr
   data["form"]=map[string][]string(req.Form)
   return data
}

func (ser *ProxyServe)logRequest(req *http.Request,ctx *goproxy.ProxyCtx,req_new *http.Request){
   req_uid:=NextUid()
  
   data_origin:=getReqLogData(req)
//   fmt.Println(data)
   data_rewrite:=getReqLogData(req_new)
   
   data:=kvType{}
   
   data["now"]=time.Now().UnixNano()
   data["session_id"]=ctx.Session
   data["user"]=ctx.UserData.(string)
   
   
   data_rewrite_change:=kvType{}
   for k,v:=range data_rewrite{
      v_e:=gob_encode(v)
      v_e_last:=gob_encode(data_origin[k])
      if(v_e!=v_e_last){
        data_rewrite_change[k]=v
      }
   }
   
   data["origin"]=gob_encode(data_origin)
   data["rewrite"]=gob_encode(data_rewrite_change)
//   fmt.Println(data)
   err:= ser.mydb.RequestTable.InsertRecovery(req_uid,data)
   
   log.Println("save_req",ctx.Session,req.URL.String(),"req_docid=",req_uid,err)
   
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
   data["now"]=time.Now().UnixNano()
   data["header"]=res.Header
   data["status"]=res.StatusCode
   data["content_length"]=res.ContentLength
//   data["cookies"]=res.Cookies()
   if(res.ContentLength<=ser.MaxResSaveLength){
	   buf:=forgetRead(&res.Body)
	   data["body"]=buf.String()
   }else{
	   data["body"]="pproxy skip"
   }
   
   err:= ser.mydb.ResponseTable.InsertRecovery(req_uid,data)
   log.Println("save_res [",req_uid,"]",err)
   if(err!=nil){
	    log.Println(err)
	    return
  }
}

func (ser *ProxyServe)GetResponseByDocid(docid uint64) (res_data kvType){
  ser.mydb.ResponseTable.Read(docid,&res_data)
//  fmt.Println(docid,res_data)
  return res_data
}
func (ser *ProxyServe)GetRequestByDocid(docid uint64) (req_data kvType){
  ser.mydb.RequestTable.Read(docid,&req_data)
//  var t kvType
//  gob_decode(req_data["origin"].(string),&t)
//  req_data["origin"]=t
//  gob_decode(req_data["rewrite"].(string),&t)
//  req_data["rewrite"]=t
//  fmt.Println(req_data)
 return req_data
}

func NewProxyServe()*ProxyServe{
   proxy:= new(ProxyServe)
   proxy.mydb=NewTieDb("./data/")
   proxy.startTime=time.Now()
   proxy.MaxResSaveLength=2*1024*1024
   
//  data:= proxy.GetRequestByDocid(14251672015029932961)
//  fmt.Println(data)
  return proxy
}


func (ser *ProxyServe)Broadcast_Req(id int64,req *http.Request,docid uint64){
  data:=make(map[string]interface{})
  data["docid"]=fmt.Sprintf("%d",docid)
  data["sid"]=id%1000
  data["host"]=req.Host
  data["path"]=req.URL.Path
  data["method"]=req.Method
  for _,client:=range ser.wsClients{
     send_req(client,data)
  }
}
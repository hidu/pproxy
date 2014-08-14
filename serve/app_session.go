package serve

import (
	"net/http"
	"time"
	 "log"
)

type clientSession struct {
	Ip               string
	Port             string
	RequestNum       int
	FirstRequestTime time.Time
	LastRequestTime  time.Time
	User             *User
}

func (ser *ProxyServe) regirestReq(req *http.Request, reqCtx *requestCtx) {
	ip := reqCtx.GetIp()
	now := time.Now()
	ser.mu.Lock()
	defer ser.mu.Unlock()
	var session *clientSession
	if client, has := ser.ProxyClients[ip]; has {
		session = client
	} else {
		session = &clientSession{
			Ip:               ip,
			RequestNum:       0,
			FirstRequestTime: now,
			LastRequestTime:  now,
		}
	}
	session.LastRequestTime=now
	session.RequestNum++
	if(ser.Debug){
		log.Println("session_debug:",session)
	}
    ser.ProxyClients[ip] = session
    
	reqCtx.ClientSession = session
}

func (ser *ProxyServe)cleanExpiredSession(){
	ser.mu.Lock()
	defer ser.mu.Unlock()
	now := time.Now()
	deleteIps:=[]string{}
	for ip,session:=range ser.ProxyClients{
	   t:=now.Sub(session.LastRequestTime)
	   if t.Minutes()>1 {
	     deleteIps=append(deleteIps,ip)
	   }
	}
	for _,ip:=range deleteIps{
	  delete(ser.ProxyClients,ip)
	  log.Println("session expired:ip=",ip)
	}
}

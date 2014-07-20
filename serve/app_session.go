package serve

import (
	"net/http"
	"time"
	//   "fmt"
)

type clientSession struct {
	Ip               string
	Port             string
	RequestNum       int
	FirstRequestTime time.Time
	LastRequestTime  time.Time
	IntervalDuration time.Duration
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
			IntervalDuration: 0,
		}
		ser.ProxyClients[ip] = session
	}
	session.IntervalDuration = now.Sub(session.LastRequestTime)
	if session.IntervalDuration.Minutes() > 10 {
		session.RequestNum = 0
	}
	session.RequestNum++
	reqCtx.ClientSession = session
}

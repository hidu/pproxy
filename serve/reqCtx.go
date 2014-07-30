package serve

import (
	"log"
	"net/url"
	"strings"
)

type requestCtx struct {
	RemoteAddr    string
	User          *User
	Docid         uint64
	IsReDo        bool
	SessionId     int64
	HasBroadcast  bool
	FormPost      *url.Values
	ClientSession *clientSession
	LogData       map[string]interface{}
}

func NewRequestCtx() *requestCtx {
	ctx := new(requestCtx)
	ctx.LogData = make(map[string]interface{})
	return ctx
}

func (ctx *requestCtx) GetIp() string {
	host_info := strings.Split(ctx.RemoteAddr, ":")
	return host_info[0]
}

func (ctx *requestCtx) PrintLog() {
	log.Println(
	           "session_id:", ctx.SessionId,
	           "reqNum:", ctx.ClientSession.RequestNum,
	           "docid:", ctx.Docid,
	            "broadcast:", ctx.HasBroadcast, 
	            "data:", ctx.LogData,
	            )
}

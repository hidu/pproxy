package serve

import (
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
}

func (ctx *requestCtx) GetIp() string {
	host_info := strings.Split(ctx.RemoteAddr, ":")
	return host_info[0]
}

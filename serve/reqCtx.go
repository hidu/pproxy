package serve

import (
	"log"
	"net/http"
	"net/url"
	"strings"
)

type requestCtx struct {
	RemoteAddr    string
	User          *User
	Docid         int
	IsRePlay      bool
	SessionId     int64
	HasBroadcast  bool
	FormPost      *url.Values
	ClientSession *clientSession
	LogData       map[string]interface{}
	OriginUrl     string
	Msg           string
}

func NewRequestCtx(ser *ProxyServe, req *http.Request) *requestCtx {
	ctx := new(requestCtx)
	ctx.LogData = make(map[string]interface{})
	ctx.FormPost = &url.Values{}
	if req != nil {
		ctx.User = getAuthorInfo(req)
		ctx.OriginUrl = req.URL.String()
		ctx.IsRePlay = len(req.Header.Get(REPLAY_FLAG)) > 0
		ctx.LogData["url"] = req.URL.String()

		ctx.RemoteAddr = req.RemoteAddr

		if _replay_addr := req.Header.Get(REPLAY_REMOTEADDR); _replay_addr != "" {
			ctx.RemoteAddr = _replay_addr
		}
		if _replay_user := req.Header.Get(REPLAY_USER_NAME); _replay_user != "" {
			ctx.User = &User{Name: _replay_user, SkipCheckPsw: true}
		}
		ctx.Docid = ser.GetNewDocid()
	}

	return ctx
}

func (ctx *requestCtx) GetIp() string {
	host_info := strings.Split(ctx.RemoteAddr, ":")
	return host_info[0]
}

func (ctx *requestCtx) PrintLog() {
	reqNum := 0
	if ctx.ClientSession != nil {
		reqNum = ctx.ClientSession.RequestNum
	}
	log.Println(
		"session_id:", ctx.SessionId,
		"remote:", ctx.RemoteAddr,
		"reqNum:", reqNum,
		"docid:", ctx.Docid,
		"uname:", ctx.User.Name,
		"broadcast:", ctx.HasBroadcast,
		"data:", ctx.LogData,
	)
}

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
	Docid         uint64
	IsReDo        bool
	SessionId     int64
	HasBroadcast  bool
	FormPost      *url.Values
	ClientSession *clientSession
	LogData       map[string]interface{}
	OriginUrl     string
	Msg           string
}

func NewRequestCtx(req *http.Request) *requestCtx {
	ctx := new(requestCtx)
	ctx.LogData = make(map[string]interface{})
	ctx.FormPost = &url.Values{}
	if req != nil {
		ctx.User = getAuthorInfo(req)
		ctx.OriginUrl = req.URL.String()
		ctx.IsReDo = len(req.Header.Get(REDO_FLAG)) > 0
		ctx.LogData["url"] = req.URL.String()

		ctx.RemoteAddr = req.RemoteAddr

		if _redo_addr := req.Header.Get(REDO_REMOTEADDR); _redo_addr != "" {
			ctx.RemoteAddr = _redo_addr
		}
		if _redo_user := req.Header.Get(REDO_USER_NAME); _redo_user != "" {
			ctx.User = &User{Name: _redo_user, SkipCheckPsw: true}
		}
		ctx.Docid = NextUid()
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
		"reqNum:", reqNum,
		"docid:", ctx.Docid,
		"uname:", ctx.User.Name,
		"broadcast:", ctx.HasBroadcast,
		"data:", ctx.LogData,
	)
}

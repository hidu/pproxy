package serve

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
	"encoding/base64"
)

type requestCtx struct {
	RemoteAddr string
	Req        *http.Request
	Rw         http.ResponseWriter

	Host string //eg www.baidu.com
	Port int    //eg 80

	User     *User
	Docid    int
	IsRePlay bool

	SessionId     int64
	HasBroadcast  bool
	FormPost      *url.Values
	ClientSession *clientSession
	LogData       map[string]interface{}
	OriginUrl     string
	Msg           string

	Tr  *http.Transport
	ser *ProxyServe
}

func NewRequestCtx(ser *ProxyServe, rw http.ResponseWriter, req *http.Request) *requestCtx {
	ctx := &requestCtx{}
	ctx.Req = req
	ctx.ser = ser
	ctx.Rw = rw

	ctx.LogData = make(map[string]interface{})
	ctx.FormPost = &url.Values{}
	ctx.init()
	ctx.Tr = &http.Transport{}
	return ctx
}

func (ctx *requestCtx) init() {
	if ctx.Req == nil {
		return
	}
	fixRequest(ctx.Req)
	req := ctx.Req
	ctx.Host, ctx.Port, _ = getHostPortFromReq(req)

	ctx.User = getAuthorInfo(req)
	ctx.FormPost = getPostData(req)

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
	ctx.Docid = ctx.ser.GetNewDocid()
	ctx.ser.regirestReq(ctx)
}

func fixRequest(req *http.Request) {
	if req.Method != "CONNECT" && !req.URL.IsAbs() {
		urlOrigin := req.URL.String()
		urlStr := "http://" + req.Host + req.URL.Path
		if req.URL.RawQuery != "" {
			urlStr += "?" + req.URL.RawQuery
		}
		var err error
		req.URL, err = url.Parse(urlStr)
		if err != nil {
			log.Println("fix url failed,originUrl:", urlOrigin, "err:", err)
			return
		}
	}
}

func (ctx *requestCtx) IsLocalRequest() bool {
	isLocalReq := ctx.Port == ctx.ser.conf.Port
	if isLocalReq {
		isLocalReq = IsLocalIp(ctx.Host)
	}
	return isLocalReq
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

func (ctx *requestCtx) RoundTrip() {
	defer ctx.PrintLog()
	
	if !ctx.ser.checkHttpAuth(ctx) {
		ctx.LogData["status"] = "login required"
		ctx.Rw.Header().Set("Proxy-Authenticate", "Basic realm=auth required")
		ctx.Rw.WriteHeader(http.StatusProxyAuthRequired)
		ctx.Rw.Write([]byte("auth required"))
		return
	}
	removeHeader(ctx.Req)
	rewrite_code := ctx.ser.reqRewrite(ctx)

	ctx.HasBroadcast = ctx.ser.Broadcast_Req(ctx)

	ctx.LogData["js_rewrite_code"] = rewrite_code
	defer ctx.saveRequestData()

	if rewrite_code != 200 && rewrite_code != 304 {
		ctx.Msg = "rewrite"
		ctx.Rw.WriteHeader(http.StatusBadGateway)
		ctx.Rw.Write([]byte("pproxy error"))
		return
	}
	resp, err := ctx.ser.RoundTrip(ctx)
	if resp == nil && err == nil {
		return
	}
	ctx.saveResponse(resp)
	if err != nil {
		ctx.Rw.WriteHeader(http.StatusBadGateway)
		ctx.Rw.Write([]byte("pproxy error"))
		return
	}
	if resp != nil {
		hijack, _ := ctx.Rw.(http.Hijacker)
		conn, _, _ := hijack.Hijack()
		defer conn.Close()
		resp.Write(conn)
		resp.Body.Close()
	}

}
func (ctx *requestCtx) DestAddr() string {
	return fmt.Sprintf("%s:%d", ctx.Host, ctx.Port)
}

func (ctx *requestCtx) saveRequestData() {
	if ctx.ser.conf.ResponseSave == ResponseSave_All ||
		(ctx.ser.conf.ResponseSave == ResponseSave_HasBroad && ctx.HasBroadcast) {
		logdata := KvType{}
		logdata["host"] = ctx.Req.Host
		logdata["schema"] = ctx.Req.URL.Scheme
		logdata["header"] = map[string][]string(ctx.Req.Header)
		logdata["url"] = ctx.Req.URL.String()
		logdata["url_origin"] = ctx.OriginUrl
		logdata["path"] = ctx.Req.URL.Path
		logdata["cookies"] = ctx.Req.Cookies()
		logdata["now"] = time.Now().Unix()
		logdata["session_id"] = ctx.SessionId
		logdata["user"] = ctx.User.Name
		logdata["client_ip"] = ctx.RemoteAddr
		logdata["method"] = ctx.Req.Method
		logdata["form_get"] = ctx.Req.URL.Query()
		logdata["replay"] = ctx.IsRePlay
		logdata["msg"] = ctx.Msg

		req_dump, err_dump := httputil.DumpRequest(ctx.Req, true)
		if err_dump != nil {
			ctx.LogData["dump"] = "dump request failed"
			req_dump = []byte("dump failed")
		}
		logdata["dump"] = base64.StdEncoding.EncodeToString(req_dump)

		logdata["form_post"] = ctx.FormPost

		err := ctx.ser.mydb.RequestTable.Set(ctx.Docid, logdata)
		if err != nil {
			log.Println("save req failed:", err)
		}
	}
}

func (ctx *requestCtx)saveResponse(res *http.Response) {
	if ctx.Docid < 1 ||res==nil{
		return
	}
	data := KvType{}
	data["session_id"] = ctx.SessionId
	data["now"] = time.Now().Unix()
	data["header"] = map[string][]string(res.Header)
	data["status"] = res.StatusCode
	data["content_length"] = res.ContentLength

	res_dump, dump_err := httputil.DumpResponse(res, false)
	if dump_err != nil {
		log.Println("dump res err", dump_err)
		res_dump = []byte("dump res failed")
	}
	data["dump"] = base64.StdEncoding.EncodeToString(res_dump)
	//   data["cookies"]=res.Cookies()

	body := []byte("pproxy skip")
	if res.ContentLength <= ctx.ser.MaxResSaveLength {
		buf := forgetRead(&res.Body)
		if res.Header.Get(Content_Encoding) == "gzip" {
			body = []byte(gzipDocode(buf))
		} else {
			body = buf.Bytes()
		}
	}
	data["body"] = base64.StdEncoding.EncodeToString(body)

	err := ctx.ser.mydb.ResponseTable.Set(ctx.Docid, data)

	log.Println("save_res", ctx.SessionId, "docid=", ctx.Docid, "body_len=", len(data["body"].(string)), err)
}

package serve

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
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

	OriginUrl string
	logData   map[interface{}]interface{}
	Msg       string

	ser           *ProxyServe
	startTime     time.Time
	timeDurations map[string]time.Duration
	hasPrint      bool
}

func NewRequestCtx(ser *ProxyServe, rw http.ResponseWriter, req *http.Request) *requestCtx {
	ctx := &requestCtx{}
	ctx.Req = req
	ctx.ser = ser
	ctx.Rw = rw
	ctx.SessionId = ser.reqNum

	ctx.logData = make(map[interface{}]interface{})
	ctx.timeDurations = make(map[string]time.Duration)

	ctx.FormPost = &url.Values{}
	ctx.init()
	ctx.startTime = time.Now()
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
	ctx.SetLog("url", req.URL.String())

	ctx.RemoteAddr = req.RemoteAddr

	if _replay_addr := req.Header.Get(REPLAY_REMOTEADDR); _replay_addr != "" {
		ctx.RemoteAddr = _replay_addr
	}
	if _replay_user := req.Header.Get(REPLAY_USER_NAME); _replay_user != "" {
		ctx.User = &User{Name: _replay_user, SkipCheckPsw: true}
	}
	ctx.Docid = ctx.getNewDocid()
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
	reqId := 0
	if ctx.ClientSession != nil {
		reqId = ctx.ClientSession.RequestNum
	}
	log.Println(
		"session_id:", ctx.SessionId,
		"remote:", ctx.RemoteAddr,
		"reqId:", reqId,
		"docid:", ctx.Docid,
		"uname:", ctx.User.Name,
		"broadcast:", ctx.HasBroadcast,
		"startTime:", ctx.startTime.Unix(),
		"timeUsed:", fmt.Sprintf("%.3fs", time.Now().Sub(ctx.startTime).Seconds()),
		"data:", ctx.logData,
		"times:", ctx.timeDurations,
	)
}

func (ctx *requestCtx) RoundTrip() {
	defer func() {
		ctx.hasPrint = true
		ctx.SetLog("logType", "defer")
		ctx.PrintLog()
	}()

	time.AfterFunc(10*time.Second, func() {
		if !ctx.hasPrint {
			ctx.SetLog("logType", "timeout10")
			ctx.PrintLog()
		}
	})

	removeHeader(ctx.Req)
	rewrite_code := ctx.ser.reqRewrite(ctx)

	ctx.HasBroadcast = ctx.ser.broadcastReq(ctx)

	ctx.SetLog("js_rewrite_code", rewrite_code)

	time.AfterFunc(1*time.Second, ctx.saveRequestData)

	if rewrite_code != 200 && rewrite_code != 304 {
		ctx.badGateway(fmt.Errorf("rewrite failed"))
		return
	}
	ctx.ser.proxy.RoundTrip(ctx)
}
func (ctx *requestCtx) badGateway(err error) {
	ctx.SetLog("errMsg", fmt.Sprintf("%s", err))
	ctx.Rw.WriteHeader(http.StatusBadGateway)
	ctx.Rw.Write([]byte("pproxy error"))
}

func (ctx *requestCtx) DestAddr() string {
	return fmt.Sprintf("%s:%d", ctx.Host, ctx.Port)
}

func (ctx *requestCtx) saveRequestData() {
	if ctx.ser.conf.ResponseSave == responseSaveAll ||
		(ctx.ser.conf.ResponseSave == responseSaveHasBroad && ctx.HasBroadcast) {
		logdata := KvType{}
		logdata["host"] = ctx.Req.Host
		logdata["schema"] = ctx.Req.URL.Scheme
		logdata["header"] = map[string][]string(ctx.Req.Header)
		logdata["url"] = ctx.Req.URL.String()
		logdata["url_origin"] = ctx.OriginUrl
		logdata["path"] = ctx.Req.URL.Path
		//		logdata["cookies"] = ctx.Req.Cookies()
		//		logdata["now"] = time.Now().Unix()
		logdata["user"] = ctx.User.Name
		logdata["client_ip"] = ctx.RemoteAddr
		logdata["method"] = ctx.Req.Method
		logdata["form_get"] = ctx.Req.URL.Query()
		logdata["replay"] = ctx.IsRePlay
		logdata["msg"] = ctx.Msg
		logdata["id"] = fmt.Sprintf("%d", ctx.Docid)

		dumpBody := false
		req_dump, err_dump := httputil.DumpRequest(ctx.Req, dumpBody)
		if err_dump != nil {
			ctx.SetLog("dumpMsg", "dump request failed")
			req_dump = []byte("dump failed")
		}
		logdata["dump"] = base64.StdEncoding.EncodeToString(req_dump)

		logdata["form_post"] = ctx.FormPost

		tb := ctx.ser.mydb.GetkvStoreTable(KV_TABLE_REQ)
		data := newStoreType(logdata)
		err := tb.Save(IntToBytes(ctx.Docid), data)
		if err != nil {
			log.Println("save req failed:", err)
		}
	}
}

func (ctx *requestCtx) saveResponse(res *http.Response) {
	if ctx.Docid < 1 || res == nil {
		return
	}
	data := KvType{}
	data["now"] = time.Now().Unix()
	data["header"] = map[string][]string(res.Header)
	data["status"] = res.StatusCode
	data["content_length"] = res.ContentLength
	data["msg"] = ctx.Msg
	data["id"] = fmt.Sprintf("%d", ctx.Docid)

	res_dump, dump_err := httputil.DumpResponse(res, false)
	if dump_err != nil {
		log.Println("dump res err", dump_err)
		res_dump = []byte("dump res failed")
	}
	data["dump"] = base64.StdEncoding.EncodeToString(res_dump)
	//   data["cookies"]=res.Cookies()

	body := []byte("pproxy skip")
	if res.Body != nil && res.ContentLength <= ctx.ser.MaxResSaveLength {
		buf := forgetRead(&res.Body)
		if res.Header.Get(contentEncoding) == "gzip" {
			body = []byte(gzipDocode(buf))
		} else {
			body = buf.Bytes()
		}
		l := int64(len(body))
		if l > ctx.ser.MaxResSaveLength {
			body = []byte(fmt.Sprintf("pproxy skip,body too large,[len=%d]", l))
		}
	}
	data["body"] = base64.StdEncoding.EncodeToString(body)

	tb := ctx.ser.mydb.GetkvStoreTable(KV_TABLE_RES)
	storeData := newStoreType(data)
	err := tb.Save(IntToBytes(ctx.Docid), storeData)

	log.Println("save_res", ctx.SessionId, "docid=", ctx.Docid, "body_len=", len(data["body"].(string)), err)
}

func (ctx *requestCtx) SetLog(k, v interface{}) {
	ctx.logData[k] = v
}
func (ctx *requestCtx) SetTimePoint(key string) {
	ctx.timeDurations[key] = time.Now().Sub(ctx.startTime)
}

func (ctx *requestCtx) getNewDocid() int {
	id_str := fmt.Sprintf("%s%d", time.Now().Format("200601021504"), ctx.ser.reqNum)
	id, err := parseDocId(id_str)
	if err == nil {
		return id
	}
	log.Println("GetNewDocid failed", id_str, err)
	return int(time.Now().UnixNano() + ctx.ser.reqNum)
}

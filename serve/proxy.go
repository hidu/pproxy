package serve

import (
	"encoding/base64"
	"github.com/hidu/goproxy"
	"github.com/hidu/goproxy/ext/auth"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

//@todo now not work
func (ser *ProxyServe) onHttpsConnect(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	//   log.Println("https:",host,ctx.Req)

	reqCtx := NewRequestCtx(ser, nil)
	reqCtx.User = &User{SkipCheckPsw: true}
	reqCtx.RemoteAddr = host
	reqCtx.Docid = 0
	reqCtx.SessionId = ctx.Session

	ser.Broadcast_Req(ctx.Req, reqCtx)
	return goproxy.OkConnect, host
}

func removeHeader(req *http.Request) {
	for k := range req.Header {
		if len(k) > 5 && k[:6] == "Proxy-" {
			req.Header.Del(k)
		}
	}
}

func (ser *ProxyServe) onRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	//	log.Println("RemoteAddr:",req.RemoteAddr,req.Header.Get("X-Wap-Proxy-Cookie"))
	reqCtx := NewRequestCtx(ser, req)
	reqCtx.SessionId = ctx.Session
	ser.regirestReq(req, reqCtx)

	defer reqCtx.PrintLog()

	if !ser.checkHttpAuth(req, reqCtx) {
		reqCtx.LogData["status"] = "login required"
		return nil, auth.BasicUnauthorized(req, "pproxy auth need")
	}

	removeHeader(req)

	post_vs := getPostData(req)
	reqCtx.FormPost = post_vs

	rewrite_code := ser.reqRewrite(req, reqCtx)
	reqCtx.LogData["js_rewrite_code"] = rewrite_code

	ctx.UserData = reqCtx

	if ser.Debug {
		req_dump_debug, _ := httputil.DumpRequest(req, false)
		log.Println("DEBUG req AFTER:\n", string(req_dump_debug), "\nurl_host:", req.URL.Host)
	}
	reqCtx.HasBroadcast = ser.Broadcast_Req(req, reqCtx)

	defer ser.saveRequestData(req, reqCtx)

	if rewrite_code != 200 && rewrite_code != 304 {
		reqCtx.Msg = "rewrite"
		return nil, goproxy.NewResponse(req, goproxy.ContentTypeText, rewrite_code, "pproxy error")
	}

	return req, nil
}

func (ser *ProxyServe) saveRequestData(req *http.Request, reqCtx *requestCtx) {
	if ser.conf.ResponseSave == ResponseSave_All || (ser.conf.ResponseSave == ResponseSave_HasBroad && reqCtx.HasBroadcast) {
		logdata := kvType{}
		logdata["host"] = req.Host
		logdata["schema"] = req.URL.Scheme
		logdata["header"] = map[string][]string(req.Header)
		logdata["url"] = req.URL.String()
		logdata["url_origin"] = reqCtx.OriginUrl
		logdata["path"] = req.URL.Path
		logdata["cookies"] = req.Cookies()
		logdata["now"] = time.Now().Unix()
		logdata["session_id"] = reqCtx.SessionId
		logdata["user"] = reqCtx.User.Name
		logdata["client_ip"] = reqCtx.RemoteAddr
		logdata["method"] = req.Method
		logdata["form_get"] = req.URL.Query()
		logdata["redo"] = reqCtx.IsReDo
		logdata["msg"] = reqCtx.Msg

		req_dump, err_dump := httputil.DumpRequest(req, true)
		if err_dump != nil {
			reqCtx.LogData["dump"] = "dump request failed"
			req_dump = []byte("dump failed")
		}
		logdata["dump"] = base64.StdEncoding.EncodeToString(req_dump)

		logdata["form_post"] = reqCtx.FormPost

		err := ser.mydb.RequestTable.Set(reqCtx.Docid, logdata)
		if err != nil {
			log.Println("save req failed:", err)
		}
	} else {
		reqCtx.Docid = 0
	}
}

func getPostData(req *http.Request) (post *url.Values) {
	post = new(url.Values)
	if strings.Contains(req.Header.Get("Content-Type"), "x-www-form-urlencoded") {
		buf := forgetRead(&req.Body)
		var body_str string
		if req.Header.Get(Content_Encoding) == "gzip" {
			body_str = gzipDocode(buf)
		} else {
			body_str = buf.String()
		}
		var err error
		*post, err = url.ParseQuery(body_str)
		if err != nil {
			log.Println("parse post err", err)
		}

	}
	return post
}

func (ser *ProxyServe) onResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp != nil {
		//resp.Header.Set("Connection", "close")
	}
	if resp == nil || resp.Request == nil {
		return resp
	}
	ser.logResponse(resp, ctx)
	return resp
}

/**
*log response if the req has log
 */
func (ser *ProxyServe) logResponse(res *http.Response, ctx *goproxy.ProxyCtx) {
	if ctx.UserData == nil {
		log.Println("err,userdata not reqid,log res skip")
		return
	}
	reqCtx := ctx.UserData.(*requestCtx)
	if reqCtx.Docid < 1 {
		return
	}
	data := kvType{}
	data["session_id"] = ctx.Session
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
	if res.ContentLength <= ser.MaxResSaveLength {
		buf := forgetRead(&res.Body)
		if res.Header.Get(Content_Encoding) == "gzip" {
			body = []byte(gzipDocode(buf))
		} else {
			body = buf.Bytes()
		}
	}
	data["body"] = base64.StdEncoding.EncodeToString(body)

	err := ser.mydb.ResponseTable.Set(reqCtx.Docid, data)

	log.Println("save_res", ctx.Session, "docid=", reqCtx.Docid, "body_len=", len(data["body"].(string)), err)
}

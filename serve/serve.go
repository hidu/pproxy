package serve

import (
	"encoding/base64"
	"fmt"
	"github.com/hidu/go-socket.io"
	"github.com/hidu/goproxy"
	"github.com/hidu/goproxy/ext/auth"
	"github.com/hidu/goutils"
	"github.com/robertkrimen/otto"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var js *otto.Otto

type ProxyServe struct {
	Goproxy *goproxy.ProxyHttpServer

	mydb      *TieDb
	ws        *socketio.SocketIOServer
	wsClients map[string]*wsClient
	startTime time.Time

	MaxResSaveLength int64

	RewriteJs string

	RewriteJsFn otto.Value
	mu          sync.RWMutex

	Debug bool

	conf      *Config
	configDir string
	hosts     configHosts

	Users        map[string]*User
	ProxyClients map[string]*clientSession
}

type kvType map[string]interface{}

func (ser *ProxyServe) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	host, port_int, err := getHostPortFromReq(req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
		log.Println("bad request,err", err)
		return
	}
	if req.Host == "p.info" || req.Host == "proxy.info" {
		ser.handleUserInfo(w, req)
		return
	}

	isLocalReq := port_int == ser.conf.Port
	if isLocalReq {
		isLocalReq = IsLocalIp(host)
	}

	if isLocalReq {
		ser.handleLocalReq(w, req)
	} else {
		ser.Goproxy.ServeHTTP(w, req)
	}
}

func (ser *ProxyServe) Start() {
	ser.Goproxy = goproxy.NewProxyHttpServer()
	ser.Goproxy.OnRequest().HandleConnectFunc(ser.onHttpsConnect)
	ser.Goproxy.OnRequest().DoFunc(ser.onRequest)
	ser.Goproxy.OnResponse().DoFunc(ser.onResponse)
	addr := fmt.Sprintf("%s:%d", "", ser.conf.Port)
	log.Println("proxy listen at ", addr)
	ser.ws_init()
	err := http.ListenAndServe(addr, ser)
	log.Println(err)
}

//@todo now not work
func (ser *ProxyServe) onHttpsConnect(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	//   log.Println("https:",host,ctx.Req)

	reqCtx := NewRequestCtx()
	reqCtx.User = &User{SkipCheckPsw: true}
	reqCtx.RemoteAddr = host
	reqCtx.Docid = 0
	reqCtx.SessionId = ctx.Session

	ser.Broadcast_Req(ctx.Req, reqCtx)
	return goproxy.OkConnect, host
}

func (ser *ProxyServe) onRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	if ser.Debug {
		req_dump_debug, _ := httputil.DumpRequest(req, false)
		log.Println("DEBUG req BEFORE:\n", string(req_dump_debug), "\nurl_full:", req.URL.String())
	}
	//	log.Println("RemoteAddr:",req.RemoteAddr,req.Header.Get("X-Wap-Proxy-Cookie"))

	reqCtx := NewRequestCtx()
	reqCtx.User = getAuthorInfo(req)
	reqCtx.IsReDo = len(req.Header.Get(REDO_FLAG)) > 0
	reqCtx.SessionId = ctx.Session

	reqCtx.LogData["url"] = req.URL.String()

	reqCtx.RemoteAddr = req.RemoteAddr
	if _redo_addr := req.Header.Get(REDO_REMOTEADDR); _redo_addr != "" {
		reqCtx.RemoteAddr = _redo_addr
	}

	if _redo_user := req.Header.Get(REDO_USER_NAME); _redo_user != "" {
		reqCtx.User = &User{Name: _redo_user, SkipCheckPsw: true}
	}

	ser.regirestReq(req, reqCtx)

	for k := range req.Header {
		if len(k) > 5 && k[:6] == "Proxy-" {
			req.Header.Del(k)
		}
	}

	if ser.conf.AuthType != AuthType_NO && !ser.checkHttpAuth(req, reqCtx) {
		log.Println("login required", req.RemoteAddr, reqCtx.User)
		return nil, auth.BasicUnauthorized(req, "pproxy auth need")
	}

	post_vs := getPostData(req)
	reqCtx.FormPost = post_vs

	rewrite_code := ser.reqRewrite(req, reqCtx)

	reqCtx.Docid = NextUid() + uint64(ctx.Session)

	ctx.UserData = reqCtx

	if ser.Debug {
		req_dump_debug, _ := httputil.DumpRequest(req, false)
		log.Println("DEBUG req AFTER:\n", string(req_dump_debug), "\nurl_host:", req.URL.Host)
	}

	reqCtx.HasBroadcast = ser.Broadcast_Req(req, reqCtx)

	if ser.conf.ResponseSave == ResponseSave_All || (ser.conf.ResponseSave == ResponseSave_HasBroad && reqCtx.HasBroadcast) {
		logdata := kvType{}
		logdata["host"] = req.Host
		logdata["header"] = map[string][]string(req.Header)
		logdata["url"] = req.URL.String()
		logdata["path"] = req.URL.Path
		logdata["cookies"] = req.Cookies()
		logdata["now"] = time.Now().Unix()
		logdata["session_id"] = ctx.Session
		logdata["user"] = reqCtx.User.Name
		logdata["client_ip"] = reqCtx.RemoteAddr
		logdata["method"] = req.Method
		logdata["form_get"] = req.URL.Query()
		logdata["redo"] = reqCtx.IsReDo

		req_dump, err_dump := httputil.DumpRequest(req, true)
		if err_dump != nil {
			log.Println("dump request failed")
			req_dump = []byte("dump failed")
		}
		logdata["dump"] = base64.StdEncoding.EncodeToString(req_dump)

		logdata["form_post"] = post_vs

		rewrite := make(map[string]string)
		url_new := req.URL.String()

		if url_new != logdata["url"] {
			rewrite["url"] = url_new
		}

		logdata["rewrite"] = rewrite

		err := ser.mydb.RequestTable.InsertRecovery(reqCtx.Docid, logdata)
		log.Println("save_req", ctx.Session, req.URL.String(), "user:", reqCtx.User.Name, "docid=", reqCtx.Docid, err, rewrite)
		if err != nil {
			log.Println(err)
			return req, nil
		}
	} else {
		reqCtx.Docid = 0
	}
	if rewrite_code != 200 {
		return nil, goproxy.NewResponse(req, goproxy.ContentTypeText, rewrite_code, "pproxy error")
	}
	return req, nil
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
		resp.Header.Set("Connection", "close")
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

	err := ser.mydb.ResponseTable.InsertRecovery(reqCtx.Docid, data)

	log.Println("save_res", ctx.Session, "docid=", reqCtx.Docid, "body_len=", len(data["body"].(string)), err)
}

func (ser *ProxyServe) GetResponseByDocid(docid uint64) (res_data kvType) {
	id, err := ser.mydb.ResponseTable.Read(docid, &res_data)
	if err != nil {
		log.Println("read res by docid failed,docid=", docid, "id=", id, err)
	}
	//  fmt.Println(docid,res_data)
	return res_data
}
func (ser *ProxyServe) GetRequestByDocid(docid uint64) (req_data kvType) {
	id, err := ser.mydb.RequestTable.Read(docid, &req_data)
	if err != nil {
		log.Println("read req by docid failed,docid=", docid, "id=", id, err)
	}
	return req_data
}

func (ser *ProxyServe) GetRewriteJsPath() string {
	return fmt.Sprintf("%s/req_rewrite_%d.js", ser.configDir, ser.conf.Port)
}

func (ser *ProxyServe) GetHostsFilePath() string {
	return fmt.Sprintf("%s/hosts_%d", ser.configDir, ser.conf.Port)
}

func (ser *ProxyServe) loadHosts() {
	ser.mu.Lock()
	defer ser.mu.Unlock()
	hosts_path := ser.GetHostsFilePath()
	log.Println("load hosts:", hosts_path)
	ser.hosts, _ = loadHosts(hosts_path)
}

func NewProxyServe(confPath string, port int) (*ProxyServe, error) {
	conf, err := LoadConfig(confPath)
	if err != nil {
		log.Println("load config faield", err)
		return nil, err
	}
	if port > 0 && port < 65535 {
		conf.Port = port
	}

	absPath, err := filepath.Abs(confPath)
	if err != nil {
		log.Println("get config path failed", confPath)
		return nil, err
	}
	GetVersion()
	os.Chdir(filepath.Dir(absPath))

	proxy := new(ProxyServe)
	proxy.configDir = filepath.Dir(absPath)
	proxy.Users, _ = loadUsers(proxy.configDir + "/users")

	proxy.conf = conf

	js = otto.New()
	jsPath := proxy.GetRewriteJsPath()

	if utils.File_exists(jsPath) {
		script, err := ioutil.ReadFile(jsPath)
		if err == nil {
			err = proxy.parseAndSaveRewriteJs(string(script))
			if err != nil {
				fmt.Println("load rewrite js failed:", err)
				return nil, err
			}
		}
	}

	proxy.loadHosts()

	proxy.mydb = NewTieDb(fmt.Sprintf("%s/%d/", conf.DataDir, conf.Port))
	proxy.startTime = time.Now()
	proxy.MaxResSaveLength = 2 * 1024 * 1024

	rand.Seed(time.Now().UnixNano())

	proxy.ProxyClients = make(map[string]*clientSession)

	//   proxy.mydb.StartGcTimer(60,store_time)
	return proxy, nil
}

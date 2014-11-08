package serve

import (
	"fmt"
	"github.com/hidu/goutils"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type ProxyServe struct {
	tripper map[string]proxyRoundTripper
	mydb    *TieDb

	wsSer *wsServer

	startTime time.Time

	MaxResSaveLength int64

	mu sync.RWMutex

	Debug bool

	conf      *Config
	configDir string
	hosts     configHosts

	Users        map[string]*User
	ProxyClients map[string]*clientSession
	reqNum       int64

	reqMod *requestModifier
}

type KvType map[string]interface{}

func (ser *ProxyServe) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	atomic.AddInt64(&ser.reqNum, 1)
	
	ctx := NewRequestCtx(ser, w, req)
	if ctx.Host == "p.info" || ctx.Host == "proxy.info" {
		ser.handleUserInfo(w, req)
		return
	}

	if ctx.IsLocalRequest() {
		ser.handleLocalReq(w, req)
	} else {
		if ser.Debug {
			req_dump_debug, _ := httputil.DumpRequest(req, req.Method == "GET")
			log.Println("DEBUG req BEFORE:\nurl_full:", req.URL.String(), "\nschema:", req.URL.Scheme, "\n", string(req_dump_debug), "\n\n")
		}
		ctx.RoundTrip()
	}
}

func (ser *ProxyServe)ServeHTTPProxy(w http.ResponseWriter, req *http.Request){
	atomic.AddInt64(&ser.reqNum, 1)
	ctx := NewRequestCtx(ser, w, req)
	ctx.RoundTrip()
}

func (ser *ProxyServe) Start() {
	addr := fmt.Sprintf("%s:%d", "", ser.conf.Port)
	fmt.Println("proxy listen at ", addr)
	ser.ws_init()
	err := http.ListenAndServe(addr, ser)
	log.Println(err)
	fmt.Println(err)
}

func (ser *ProxyServe) GetNewDocid() int {
	id_str := fmt.Sprintf("%s%d", time.Now().Format("200601021504"), ser.reqNum)
	id, err := parseDocId(id_str)
	if err == nil {
		return id
	}
	log.Println("GetNewDocid failed", id_str, err)
	return int(time.Now().UnixNano() + ser.reqNum)
}

func (ser *ProxyServe) GetResponseByDocid(docid int) (res_data KvType) {
	res_data, err := ser.mydb.ResponseTable.GetByKey(docid)
	if err != nil {
		log.Println("read res by docid failed,docid=", docid, err)
	}
	//  fmt.Println(docid,res_data)
	return res_data
}
func (ser *ProxyServe) GetRequestByDocid(docid int) (req_data KvType) {
	req_data, err := ser.mydb.RequestTable.GetByKey(docid)
	if err != nil {
		log.Println("read req by docid failed,docid=", docid, err)
	}
	return req_data
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
	setupLog(conf.DataDir, conf.Port)

	proxy := new(ProxyServe)
	proxy.configDir = filepath.Dir(absPath)
	proxy.Users, _ = loadUsers(proxy.configDir + "/users")

	conf.FileDir, _ = filepath.Abs(conf.FileDir)

	proxy.conf = conf

	proxy.reqMod = NewRequestModifier(proxy)
	err = proxy.reqMod.loadAllJs()
	if err != nil {
		return nil, err
	}

	proxy.loadHosts()

	proxy.mydb = NewTieDb(fmt.Sprintf("%s/%d/", conf.DataDir, conf.Port), conf.DataStoreDay)
	proxy.startTime = time.Now()
	proxy.MaxResSaveLength = 2 * 1024 * 1024

	rand.Seed(time.Now().UnixNano())

	proxy.ProxyClients = make(map[string]*clientSession)
	proxy.tripper = make(map[string]proxyRoundTripper)

	proxy.tripper["default"] = RoundTrip_Default
	proxy.tripper["upgrade"] = RoundTrip_Upgrade

	utils.SetInterval(func() {
		proxy.cleanExpiredSession()
	}, 60)

	//   proxy.mydb.StartGcTimer(60,store_time)
	return proxy, nil
}

func (ser *ProxyServe) RoundTrip(ctx *requestCtx) (resp *http.Response, err error) {
	rtName := "default"
	if ctx.Req.Header.Get("Upgrade") != "" {
		rtName = "Upgrade"
	}
	if rt, has := ser.tripper[rtName]; has {
		return rt(ctx)
	}
	return nil, fmt.Errorf("unknow roundTrip")
}

func setupLog(dataDir string, port int) {
	logPath := fmt.Sprintf("%s/%d.log", dataDir, port)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Println("create log file failed [", logPath, "]", err)
		os.Exit(2)
	}
	log.SetOutput(logFile)

	utils.SetInterval(func() {
		if !utils.File_exists(logPath) {
			logFile.Close()
			logFile, _ = os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
			log.SetOutput(logFile)
		}
	}, 30)
}

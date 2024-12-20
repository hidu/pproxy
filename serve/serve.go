package serve

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hidu/goutils/fs"
	"github.com/hidu/goutils/time_util"
)

type ProxyServe struct {
	mydb  *kvStore
	proxy *HttpProxy

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

type KvType map[string]any

func (ser *ProxyServe) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	atomic.AddInt64(&ser.reqNum, 1)

	// 	reqDump, _ := httputil.DumpRequest(req, true)
	//    fmt.Println("req dump:\n",string(reqDump))

	ctx := NewRequestCtx(ser, w, req)
	if ctx.Host == "p.info" || ctx.Host == "pproxy.info" {
		ser.handleUserInfo(w, req)
		return
	}

	if ctx.Host == "pproxy.man" || ctx.Host == "pproxy.com" || ctx.IsLocalRequest() {
		ser.handleLocalReq(w, req)
	} else {
		if ser.Debug {
			reqDumpDebug, _ := httputil.DumpRequest(req, req.Method == "GET")
			log.Println("DEBUG req BEFORE:\nurl_full:", req.URL.String(), "\nschema:", req.URL.Scheme, "\n", string(reqDumpDebug), "\n\n")
		}
		if !ser.checkHTTPAuth(ctx) {
			ctx.SetLog("msg", "login required")
			ctx.Rw.Header().Set("Proxy-Authenticate", "Basic realm=auth required")
			ctx.Rw.WriteHeader(http.StatusProxyAuthRequired)
			ctx.Rw.Write([]byte("auth required"))
			return
		}
		ctx.RoundTrip()
	}
}

// for replay
func (ser *ProxyServe) ServeHTTPProxy(w http.ResponseWriter, req *http.Request) {
	atomic.AddInt64(&ser.reqNum, 1)
	ctx := NewRequestCtx(ser, w, req)
	ctx.RoundTrip()
}

func (ser *ProxyServe) Start() {
	addr := fmt.Sprintf("%s:%d", "", ser.conf.Port)
	fmt.Println("proxy listen at ", addr)
	defer log.Println("pproxy exit")

	ser.wsInit()

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		err := http.ListenAndServe(addr, ser)
		log.Println(err)
		fmt.Println(err)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		ser.startAdmin()
		wg.Done()
	}()

	wg.Wait()
}

func (ser *ProxyServe) startAdmin() {
	if ser.conf.Port == ser.conf.AdminPort {
		return
	}
	addr := fmt.Sprintf(":%d", ser.conf.AdminPort)
	fmt.Println("admin http service listen at ", addr)
	httpSer := http.NewServeMux()
	httpSer.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		ser.handleLocalReq(w, req)
	})
	http.ListenAndServe(addr, httpSer)
}

func (ser *ProxyServe) getResponseByDocid(docid int) (resData *StoreType, err error) {
	tb := ser.mydb.GetkvStoreTable(KV_TABLE_RES)
	return tb.Get(IntToBytes(docid))
}

func (ser *ProxyServe) getRequestByDocid(docid int) (reqData *StoreType, err error) {
	tb := ser.mydb.GetkvStoreTable(KV_TABLE_REQ)
	return tb.Get(IntToBytes(docid))
}

func (ser *ProxyServe) getHostsFilePath() string {
	return fmt.Sprintf("%s/hosts_%d", ser.configDir, ser.conf.Port)
}

func (ser *ProxyServe) loadHosts() {
	ser.mu.Lock()
	defer ser.mu.Unlock()
	hostsPath := ser.getHostsFilePath()
	log.Println("load hosts:", hostsPath)
	ser.hosts, _ = loadHosts(hostsPath)
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

	dbPath := fmt.Sprintf("%s/%d.db", conf.DataDir, conf.Port)

	// 	proxy.mydb = NewTieDb(fmt.Sprintf("%s/%d/", conf.DataDir, conf.Port), conf.DataStoreDay)
	proxy.mydb, err = newKvStore(dbPath)
	if err != nil {
		log.Fatalln("init db failed", err)
	}

	proxy.startTime = time.Now()
	proxy.MaxResSaveLength = 2 * 1024 * 1024

	rand.Seed(time.Now().UnixNano())

	proxy.ProxyClients = make(map[string]*clientSession)
	proxy.proxy = NewHttpProxy(proxy)

	time_util.SetInterval(func() {
		proxy.cleanExpiredSession()
	}, 60)
	proxy.mydb.StartGcTimer(60, int64(conf.DataStoreDay*86400))
	return proxy, nil
}

func setupLog(dataDir string, port int) {
	logPath := fmt.Sprintf("%s/%d.log", dataDir, port)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Println("create log file failed [", logPath, "]", err)
		os.Exit(2)
	}
	log.SetOutput(logFile)

	time_util.SetInterval(func() {
		if !fs.FileExists(logPath) {
			logFile.Close()
			logFile, _ = os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
			log.SetOutput(logFile)
		}
	}, 30)
}

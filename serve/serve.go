package serve

import (
	"fmt"
	"github.com/googollee/go-socket.io"
	"github.com/hidu/goproxy"

	"github.com/hidu/goutils"
	"github.com/robertkrimen/otto"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"

	"os"
	"path/filepath"

	"sync"
	"time"
)

var js *otto.Otto

type ProxyServe struct {
	Goproxy *goproxy.ProxyHttpServer

	mydb      *TieDb
	ws        *socketio.Server
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
	fmt.Println(err)
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
	setupLog(conf.DataDir, conf.Port)

	proxy.loadHosts()

	proxy.mydb = NewTieDb(fmt.Sprintf("%s/%d/", conf.DataDir, conf.Port))
	proxy.startTime = time.Now()
	proxy.MaxResSaveLength = 2 * 1024 * 1024

	rand.Seed(time.Now().UnixNano())

	proxy.ProxyClients = make(map[string]*clientSession)

	//   proxy.mydb.StartGcTimer(60,store_time)
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

	utils.SetInterval(func() {
		if !utils.File_exists(logPath) {
			logFile.Close()
			logFile, _ = os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
			log.SetOutput(logFile)
		}
	}, 30)
}

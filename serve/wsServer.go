package serve

import (
	"fmt"
	"github.com/hidu/goutils"
	"github.com/googollee/go-socket.io"
	"log"
	"net/http"
	"net/url"
	"sync"
)

type wsServer struct {
	clients  map[string]*wsClient
	server   *socketio.SocketIOServer
	mu       sync.RWMutex
	proxySer *ProxyServe
}

func (ser *ProxyServe) wsInit() {
	ser.wsSer = newWsServer(ser)
}

func newWsServer(ser *ProxyServe) *wsServer {
	wsSer := &wsServer{
		clients:  make(map[string]*wsClient),
		proxySer: ser,
	}
	var err error
	wsSer.server = socketio.NewSocketIOServer(&socketio.Config{})
	if err != nil {
		log.Fatal(err)
	}
	wsSer.init()
	return wsSer
}

func (wsSer *wsServer) init() {
	wsSer.server.On("connect", func(ns *socketio.NameSpace) {
		wsSer.mu.Lock()
		defer wsSer.mu.Unlock()
		wsSer.clients[ns.Id()] = &wsClient{ns: ns, user: "guest"}

		log.Println("ws connected", ns.Session.Request.RemoteAddr, ns.Id(), "ws_client_num:", len(wsSer.clients))
	})
	wsSer.server.On("disconnect", func(ns *socketio.NameSpace) {
		wsSer.remove(ns.Id())
		log.Println("ws disconnect", ns.Session.Request.RemoteAddr, ns.Id(), "ws_client_num:", len(wsSer.clients))
	})
	wsSer.server.On("error", func(ns *socketio.NameSpace, err error) {
		log.Println("ws error:", err)
	})
	wsSer.server.On("get_response", wsSer.getResponse)
	wsSer.server.On("client_filter", wsSer.saveFilter)

	utils.SetInterval(func() {
		wsSer.broadcast("hello", "hello", false)
	}, 120)
}

func (wsSer *wsServer) remove(id string) {
	wsSer.mu.Lock()
	defer wsSer.mu.Unlock()
	if _, has := wsSer.clients[id]; has {
		delete(wsSer.clients, id)
	}
}

func (wsSer *wsServer) broadProxyClientNum() {
	wsSer.broadcast("user_num", len(wsSer.proxySer.ProxyClients), false)
}

/**
*https://github.com/googollee/go-socket.io
 */
func (wsSer *wsServer) getResponse(ns *socketio.NameSpace, docidStr string) {
	docid, uintParseErr := parseDocID(docidStr)
	if uintParseErr != nil {
		log.Println("parse str2int failed", docidStr, uintParseErr)
		return
	}
	log.Println("receive docid", docid, ns.Session.Request.RemoteAddr)
	req, _ := wsSer.proxySer.getRequestByDocid(docid)
	res, _ := wsSer.proxySer.getResponseByDocid(docid)
	if wsSer.proxySer.Debug {
		fmt.Println("req:\n", req, "\n==========\n")
		fmt.Println("res:\n", res, "\n==========\n")
	}
	//	delete(req,"header")
	data := make(map[string]interface{})
	data["req"] = nil
	data["res"] = nil
	if req != nil {
		data["req"] = req.Data
	}
	if res != nil {
		data["res"] = res.Data
	}
	wsSer.send(ns, "res", data, true)
}

func (wsSer *wsServer) saveFilter(ns *socketio.NameSpace, formData string) {
	m, err := url.ParseQuery(formData)
	if err != nil {
		log.Println("parse filter data err", err)
		return
	}
	wsSer.mu.Lock()
	defer wsSer.mu.Unlock()
	if nsClient, has := wsSer.clients[ns.Id()]; has {
		nsClient.filterIP = parseURLInputAsSlice(m.Get("client_ip"))
		nsClient.filterHideExt = m["hide"]
		nsClient.filterURL = parseURLInputAsSlice(m.Get("url_match"))
		nsClient.filterURLHide = parseURLInputAsSlice(m.Get("hide_url"))
		nsClient.filterUser = parseURLInputAsSlice(m.Get("user"))

		loginUser, isLogin := wsSer.proxySer.web_checkLogin(ns.Session.Request)
		if isLogin {
			nsClient.LoginUser = loginUser
		}
	} else {
		log.Println("ws_saveFilter failed,ws not exists")
	}
}

var nnnn int

func (wsSer *wsServer) send(ns *socketio.NameSpace, msgName string, data interface{}, encode bool) {
	wsSer.mu.Lock()

	defer func(ns *socketio.NameSpace) {
		wsSer.mu.Unlock()
		if e := recover(); e != nil {
			log.Println("ws_send failed", e, ns.Session.Request.RemoteAddr, "msgName:", msgName, "client:", len(wsSer.clients))
			wsSer.remove(ns.Id())
		}
	}(ns)
	var err error
	encode = false
	if encode {
		err = ns.Emit(msgName, dataEncode(data))
	} else {
		err = ns.Emit(msgName, data)
	}
	if err != nil {
		log.Println("emit_failed", msgName, err)
	}
}

func (wsSer *wsServer) broadcastReq(req *http.Request, reqCtx *requestCtx, data interface{}) bool {
	wsSer.mu.RLock()
	defer wsSer.mu.RUnlock()

	hasSend := false
	for _, client := range wsSer.clients {
		if wsSer.proxySer.conf.SessionView == sessionViewIPOrUser && len(client.filterIP) == 0 && len(client.filterUser) == 0 {
			continue
		}

		if reqCtx.User.Name != "" && len(client.filterUser) < 1 {
			continue
		}

		if client.checkFilter(req, reqCtx) {
			go wsSer.send(client.ns, "req", data, true)
			hasSend = true
		}
	}
	return hasSend
}

func (wsSer *wsServer) broadcast(name string, data interface{}, encode bool) {
	wsSer.mu.RLock()
	defer wsSer.mu.RUnlock()
	for _, client := range wsSer.clients {
		go wsSer.send(client.ns, name, data, encode)
	}
}

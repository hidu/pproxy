package serve

import (
	"fmt"
	"github.com/hidu/goutils"
	"gopkg.in/hidu/go-socket.io.v1"
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

func (ser *ProxyServe) ws_init() {
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
	wsSer.server.On("get_response", wsSer.get_response)
	wsSer.server.On("client_filter", wsSer.save_filter)

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
func (wsSer *wsServer) get_response(ns *socketio.NameSpace, docid_str string) {
	docid, uint_parse_err := parseDocId(docid_str)
	if uint_parse_err != nil {
		log.Println("parse str2int failed", docid_str, uint_parse_err)
		return
	}
	log.Println("receive docid", docid, ns.Session.Request.RemoteAddr)
	req := wsSer.proxySer.GetRequestByDocid(docid)
	res := wsSer.proxySer.GetResponseByDocid(docid)
	if wsSer.proxySer.Debug {
		fmt.Println("req:\n", req, "\n==========\n")
		fmt.Println("res:\n", res, "\n==========\n")
	}
	//	delete(req,"header")
	data := make(map[string]interface{})
	data["req"] = req
	data["res"] = res
	wsSer.send(ns, "res", data, true)
}

func (wsSer *wsServer) save_filter(ns *socketio.NameSpace, form_data string) {
	m, err := url.ParseQuery(form_data)
	if err != nil {
		log.Println("parse filter data err", err)
		return
	}
	wsSer.mu.Lock()
	defer wsSer.mu.Unlock()
	if nsClient, has := wsSer.clients[ns.Id()]; has {
		nsClient.filter_ip = parseUrlInputAsSlice(m.Get("client_ip"))
		nsClient.filter_hide_ext = m["hide"]
		nsClient.filter_url = parseUrlInputAsSlice(m.Get("url_match"))
		nsClient.filter_url_hide = parseUrlInputAsSlice(m.Get("hide_url"))
		nsClient.filter_user = parseUrlInputAsSlice(m.Get("user"))

		loginUser, isLogin := wsSer.proxySer.web_checkLogin(ns.Session.Request)
		if isLogin {
			nsClient.LoginUser = loginUser
		}
	} else {
		log.Println("ws_save_filter failed,ws not exists")
	}
}

var nnnn int = 0

func (wsSer *wsServer) send(ns *socketio.NameSpace, msg_name string, data interface{}, encode bool) {
	wsSer.mu.Lock()

	defer func(ns *socketio.NameSpace) {
		wsSer.mu.Unlock()
		if e := recover(); e != nil {
			log.Println("ws_send failed", e, ns.Session.Request.RemoteAddr, "msg_name:", msg_name, "client:", len(wsSer.clients))
			wsSer.remove(ns.Id())
		}
	}(ns)
	var err error
	if encode {
		err = ns.Emit(msg_name, gob_encode(data))
	} else {
		err = ns.Emit(msg_name, data)
	}
	if err != nil {
		log.Println("emit ", msg_name, " failed", err)
	}
}

func (wsSer *wsServer) broadcastReq(req *http.Request, reqCtx *requestCtx, data interface{}) bool {
	wsSer.mu.RLock()
	defer wsSer.mu.RUnlock()

	hasSend := false
	for _, client := range wsSer.clients {
		if wsSer.proxySer.conf.SessionView == SessionView_IP_OR_USER && len(client.filter_ip) == 0 && len(client.filter_user) == 0 {
			continue
		}

		if reqCtx.User.Name != "" && len(client.filter_user) < 1 {
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

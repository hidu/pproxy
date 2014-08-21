package serve

import (
	"fmt"
	"github.com/googollee/go-socket.io"
	"github.com/hidu/goutils"
	"log"
	"net/http"
	"net/url"
	"sync"
)

type wsServer struct {
	clients  map[string]*wsClient
	server   *socketio.Server
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
	wsSer.server, err = socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	wsSer.init()
	return wsSer
}

func (wsSer *wsServer) init() {
	wsSer.server.On("connection", func(ns socketio.Socket) {
		log.Println("ws connected", ns.Request().RemoteAddr, ns.Id())
		wsSer.mu.Lock()
		defer wsSer.mu.Unlock()
		wsSer.clients[ns.Id()] = &wsClient{ns: ns, user: "guest"}
	})
	wsSer.server.On("disconnection", func(ns socketio.Socket) {
		log.Println("ws disconnect", ns.Request().RemoteAddr, ns.Id())
		wsSer.remove(ns.Id())
	})
	wsSer.server.On("error", func(ns socketio.Socket, err error) {
		log.Println("ws error:", err)
	})
	wsSer.server.On("get_response", wsSer.get_response)
	wsSer.server.On("client_filter", wsSer.save_filter)

	utils.SetInterval(func() {
		wsSer.broadcastHello()
	}, 120)
}

func (wsSer *wsServer) remove(id string) {
	wsSer.mu.Lock()
	defer wsSer.mu.Unlock()
	if _, has := wsSer.clients[id]; has {
		delete(wsSer.clients, id)
	}
}

/**
*https://github.com/googollee/go-socket.io
 */
func (wsSer *wsServer) get_response(ns socketio.Socket, docid_str string) {
	docid, uint_parse_err := parseDocId(docid_str)
	if uint_parse_err != nil {
		log.Println("parse str2int failed", docid_str, uint_parse_err)
		return
	}
	log.Println("receive docid", docid)
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

func (wsSer *wsServer) save_filter(ns socketio.Socket, form_data string) {
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

		loginUser, isLogin := wsSer.proxySer.web_checkLogin(ns.Request())
		if isLogin {
			nsClient.LoginUser = loginUser
		}
	} else {
		log.Println("ws_save_filter failed,ws not exists")
	}
}

var nnnn int = 0

func (wsSer *wsServer) send(ns socketio.Socket, msg_name string, data interface{}, encode bool) {
	wsSer.mu.Lock()

	defer func(ns socketio.Socket) {
		wsSer.mu.Unlock()
		if e := recover(); e != nil {
			log.Println("ws_send failed", e)
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

func (wsSer *wsServer) broadcastHello() {
	for _, client := range wsSer.clients {
		wsSer.send(client.ns, "hello", "hidu", false)
	}
}

func (wsSer *wsServer) broadcastReq(req *http.Request, reqCtx *requestCtx, data interface{}) bool {
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

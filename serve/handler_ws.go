package serve

import (
	"github.com/hidu/go-socket.io"
	"log"
	"net/url"
	"strconv"
)

type wsClient struct {
	ns              *socketio.NameSpace
	user            string
	filter_user     []string
	filter_ip       []string
	filter_hide_ext []string
	filter_url      []string
	filter_url_hide []string
	LoginUser       *User
}

func (ser *ProxyServe) ws_init() {
	sock_config := &socketio.Config{HeartbeatTimeout: 2, ClosingTimeout: 4}

	ser.ws = socketio.NewSocketIOServer(sock_config)

	ser.wsClients = make(map[string]*wsClient)
	ser.ws.On("connect", func(ns *socketio.NameSpace) {
		log.Println("ws connected", ns.Session.Request.RemoteAddr, ns.Id(), " in channel ", ns.Endpoint())
		ser.mu.Lock()
		defer ser.mu.Unlock()
		ser.wsClients[ns.Id()] = &wsClient{ns: ns, user: "guest"}
	})
	ser.ws.On("disconnect", func(ns *socketio.NameSpace) {
		log.Println("ws disconnect", ns.Session.Request.RemoteAddr, ns.Id(), " in channel ", ns.Endpoint())
		ser.mu.Lock()
		defer ser.mu.Unlock()
		if _, has := ser.wsClients[ns.Id()]; has {
			delete(ser.wsClients, ns.Id())
		}
	})
	ser.ws.On("get_response", ser.ws_get_response)
	ser.ws.On("client_filter", ser.ws_save_filter)
}

/**
*https://github.com/googollee/go-socket.io
 */
func (ser *ProxyServe) ws_get_response(ns *socketio.NameSpace, docid_str string) {
	docid, err_int := strconv.ParseUint(docid_str, 10, 64)
	if err_int != nil {
		log.Println("parse str2int failed", err_int, docid_str)
	}
	log.Println("receive docid", docid)
	req := ser.GetRequestByDocid(docid)
	res := ser.GetResponseByDocid(docid)
	//	fmt.Println(req)
	data := make(map[string]interface{})
	data["req"] = req
	data["res"] = res
	err := ns.Emit("res", data)
	if err != nil {
		log.Println("ns error:", err)
	}
}

func (ser *ProxyServe) ws_save_filter(ns *socketio.NameSpace, form_data string) {
	m, err := url.ParseQuery(form_data)
	if err != nil {
		log.Println("parse filter data err", err)
		return
	}
	ser.mu.Lock()
	defer ser.mu.Unlock()
	nsClient := ser.wsClients[ns.Id()]
	nsClient.filter_ip = parseUrlInputAsSlice(m.Get("client_ip"))
	nsClient.filter_hide_ext = m["hide"]
	nsClient.filter_url = parseUrlInputAsSlice(m.Get("url_match"))
	nsClient.filter_url_hide = parseUrlInputAsSlice(m.Get("hide_url"))
	nsClient.filter_user = parseUrlInputAsSlice(m.Get("user"))

	loginUser, isLogin := ser.web_checkLogin(ns.Session.Request)
	if isLogin {
		nsClient.LoginUser = loginUser
	}
}

func send_req(client *wsClient, data map[string]interface{}) {
	err := client.ns.Emit("req", data)
	if err != nil {
		log.Println("emit req failed", err)
	}
}

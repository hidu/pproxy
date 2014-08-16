package serve

import (
	"fmt"
	"github.com/googollee/go-socket.io"
	"log"
	"net/url"
)

type wsClient struct {
	ns              socketio.Socket
	user            string
	filter_user     []string
	filter_ip       []string
	filter_hide_ext []string
	filter_url      []string
	filter_url_hide []string
	LoginUser       *User
}

func (ser *ProxyServe) ws_init() {
	var err error
	ser.ws, err = socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	ser.wsClients = make(map[string]*wsClient)
	ser.ws.On("connection", func(ns socketio.Socket) {
		log.Println("ws connected", ns.Request().RemoteAddr, ns.Id())
		ser.mu.Lock()
		defer ser.mu.Unlock()
		ser.wsClients[ns.Id()] = &wsClient{ns: ns, user: "guest"}
	})
	ser.ws.On("disconnection", func(ns socketio.Socket) {
		log.Println("ws disconnect", ns.Request().RemoteAddr, ns.Id())
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
func (ser *ProxyServe) ws_get_response(ns socketio.Socket, docid_str string) {
	docid, uint_parse_err := parseDocId(docid_str)
	if uint_parse_err != nil {
		log.Println("parse str2int failed", docid_str, uint_parse_err)
		return
	}
	log.Println("receive docid", docid)
	req := ser.GetRequestByDocid(docid)
	res := ser.GetResponseByDocid(docid)
	if ser.Debug {
		fmt.Println("req:\n", req, "\n==========\n")
		fmt.Println("res:\n", res, "\n==========\n")
	}
	//	delete(req,"header")
	data := make(map[string]interface{})
	data["req"] = req
	data["res"] = res

	err := ns.Emit("res", gob_encode(data))
	if err != nil {
		log.Println("ns error:", err)
	}
}

func (ser *ProxyServe) ws_save_filter(ns socketio.Socket, form_data string) {
	m, err := url.ParseQuery(form_data)
	if err != nil {
		log.Println("parse filter data err", err)
		return
	}
	ser.mu.Lock()
	defer ser.mu.Unlock()
	if nsClient, has := ser.wsClients[ns.Id()]; has {
		nsClient.filter_ip = parseUrlInputAsSlice(m.Get("client_ip"))
		nsClient.filter_hide_ext = m["hide"]
		nsClient.filter_url = parseUrlInputAsSlice(m.Get("url_match"))
		nsClient.filter_url_hide = parseUrlInputAsSlice(m.Get("hide_url"))
		nsClient.filter_user = parseUrlInputAsSlice(m.Get("user"))

		loginUser, isLogin := ser.web_checkLogin(ns.Request())
		if isLogin {
			nsClient.LoginUser = loginUser
		}
	} else {
		log.Println("ws_save_filter failed,ws not exists")
	}
}

func send_req(client *wsClient, data map[string]interface{}) {
	err := client.ns.Emit("req", gob_encode(data))
	if err != nil {
		log.Println("emit req failed", err)
	}
}

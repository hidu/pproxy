package serve

import (
	"github.com/googollee/go-socket.io"
	"github.com/hidu/goutils"
	"log"
	"net/http"
	"strings"
	"text/template"
	//  "fmt"
	"strconv"
)

/**
*https://github.com/googollee/go-socket.io
 */
func (ser *ProxyServe) client_get_response(ns *socketio.NameSpace, docid_str string) {
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
func send_req(client *wsClient, data map[string]interface{}) {
	client.ns.Emit("req", data)
}

func (ser *ProxyServe) initWs() {
	sock_config := &socketio.Config{HeartbeatTimeout: 2, ClosingTimeout: 4}
	ser.ws = socketio.NewSocketIOServer(sock_config)
	ser.wsClients = make(map[string]*wsClient)
	ser.ws.On("connect", func(ns *socketio.NameSpace) {
		log.Println("ws connected", ns.Id(), " in channel ", ns.Endpoint())
		ser.wsClients[ns.Id()] = &wsClient{ns: ns, user: "guest"}
	})
	ser.ws.On("disconnect", func(ns *socketio.NameSpace) {
		log.Println("ws disconnect", ns.Id(), " in channel ", ns.Endpoint())
		if _, has := ser.wsClients[ns.Id()]; has {
			delete(ser.wsClients, ns.Id())
		}
	})
	ser.ws.On("get_response", ser.client_get_response)
	//	ser.ws.On("req", new_req)
}

func (ser *ProxyServe) handleLocalReq(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/socket.io/1/") {
		ser.ws.ServeHTTP(w, req)
		return
	}
	if req.Method == "GET" {
		if strings.HasPrefix(req.URL.Path, "/res/") {
			goutils.DefaultResource.HandleStatic(w, req, req.URL.Path)
		} else {
			msg := goutils.DefaultResource.Load("/res/tpl/index.html")
			tpl, _ := template.New("page").Parse(string(msg))
			values := make(map[string]string)
			values["host"] = req.Host
			values["title"] = ""
			values["version"] = "0.1"
			tpl.Execute(w, values)
		}
	}
}

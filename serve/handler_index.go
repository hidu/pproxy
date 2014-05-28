package serve

import (
	"github.com/googollee/go-socket.io"
	"github.com/hidu/goutils"
	"log"
	"net/http"
	"strings"
	"text/template"
//	  "fmt"
	"strconv"
	"net/url"
	"bytes"
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
func (ser *ProxyServe) client_filter(ns *socketio.NameSpace, form_data string) {
	 m,err:= url.ParseQuery(form_data)
	 if(err!=nil){
	    log.Println("parse filter data err",err)
	    return
	 }
	 ser.mu.Lock()
	 defer ser.mu.Unlock()
 	 nsClient:=ser.wsClients[ns.Id()]
 	 nsClient.filter_client_ip=strings.TrimSpace(m.Get("client_ip"))
 	 nsClient.filter_hide=m["hide"]
 	 nsClient.filter_url=strings.Split(strings.Replace(m.Get("url_match")," ","",-1),"|")
 	 user:=strings.TrimSpace(m.Get("user"))
 	 nsClient.user=user
 	 if(user==""){
 	 	nsClient.user="guest"
 	 }
}

func send_req(client *wsClient, data map[string]interface{}) {
	err:=client.ns.Emit("req", data)
	if(err!=nil){
	  log.Println("emit req failed",err)
	}
}

func (ser *ProxyServe) initWs() {
	sock_config := &socketio.Config{HeartbeatTimeout: 2, ClosingTimeout: 4}
	ser.ws = socketio.NewSocketIOServer(sock_config)
	ser.wsClients = make(map[string]*wsClient)
	ser.ws.On("connect", func(ns *socketio.NameSpace) {
		log.Println("ws connected", ns.Id(), " in channel ", ns.Endpoint())
		ser.mu.Lock()
		defer ser.mu.Unlock()
		ser.wsClients[ns.Id()] = &wsClient{ns: ns, user: "guest"}
	})
	ser.ws.On("disconnect", func(ns *socketio.NameSpace) {
		log.Println("ws disconnect", ns.Id(), " in channel ", ns.Endpoint())
		ser.mu.Lock()
		defer ser.mu.Unlock()
		if _, has := ser.wsClients[ns.Id()]; has {
			delete(ser.wsClients, ns.Id())
		}
	})
	ser.ws.On("get_response", ser.client_get_response)
	ser.ws.On("client_filter", ser.client_filter)
}

func (ser *ProxyServe) handleLocalReq(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/socket.io/1/") {
		ser.ws.ServeHTTP(w, req)
		return
	}
	
	if strings.HasPrefix(req.URL.Path, "/res/") {
		goutils.DefaultResource.HandleStatic(w, req, req.URL.Path)
	} else if(req.URL.Path=="/") {
		values := make(map[string]interface{})
		html:=render_html("network.html",values,true)
		w.Write([]byte(html))
	}else if(req.URL.Path=="/config") {
	  if(req.Method=="GET"){
		values := make(map[string]interface{})
		values["rewriteJs"]=ser.RewriteJs
		values["rewriteJsPath"]=ser.RewriteJsPath
		html:=render_html("config.html",values,true)
		w.Write([]byte(html))
	  }else if(req.Method=="POST"){
	     ser.handleConfig(w,req)
	  }
	}else{
	  http.NotFound(w,req)
	}
}

func (ser *ProxyServe)handleConfig(w http.ResponseWriter,req *http.Request){
	 ser.mu.Lock()
	 defer ser.mu.Unlock()
 	 jsStr:=req.PostFormValue("js")
 	 err:=ser.parseAndSaveRewriteJs(jsStr)
 	 if(err==nil){
 	   if(goutils.File_exists(ser.RewriteJsPath)){
 	      err=goutils.File_put_contents(ser.RewriteJsPath,[]byte(jsStr))
 	      log.Println("save rewritejs ",ser.RewriteJsPath,err)
 	   } 
 	   w.Write([]byte("<html>save suc<script>setTimeout(function(){location.href='/config'},1000)</script></html>"))
 	 }else{
 	   w.Write([]byte("save failed,js err:"+err.Error()))
 	 }
}

func render_html(fileName string,values map[string]interface{},layout bool) string{
	html := goutils.DefaultResource.Load("/res/tpl/"+fileName)
	tpl, _ := template.New("page").Parse(string(html))
	var bf []byte
	w:=bytes.NewBuffer(bf)
	tpl.Execute(w, values)
	body:=w.String()
	if(layout){
	   values["body"]=body
	   values["title"] = ""
	   values["version"] = "0.2"
	   return render_html("layout.html",values,false)
	}
	return body
}

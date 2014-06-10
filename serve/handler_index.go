package serve

import (
    "github.com/googollee/go-socket.io"
    "github.com/hidu/goutils"
    "log"
    "net"
    "net/http"
    "strings"
    "text/template"
    //	  "fmt"
    "bytes"
    "encoding/base64"
    "html"
    "net/url"
    "strconv"
    "path/filepath"
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
    m, err := url.ParseQuery(form_data)
    if err != nil {
        log.Println("parse filter data err", err)
        return
    }
    ser.mu.Lock()
    defer ser.mu.Unlock()
    nsClient := ser.wsClients[ns.Id()]
    nsClient.filter_client_ip = strings.TrimSpace(m.Get("client_ip"))
    nsClient.filter_hide = m["hide"]
    nsClient.filter_url = strings.Split(strings.Replace(m.Get("url_match"), " ", "", -1), "|")
    user := strings.TrimSpace(m.Get("user"))
    nsClient.user = user
    if user == "" {
        nsClient.user = "guest"
    }
}

func send_req(client *wsClient, data map[string]interface{}) {
    err := client.ns.Emit("req", data)
    if err != nil {
        log.Println("emit req failed", err)
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

    values := make(map[string]interface{})
    values["title"]=ser.conf.Title
    values["notice"]=ser.conf.Notice
    
    if strings.HasPrefix(req.URL.Path, "/res/") {
        goutils.DefaultResource.HandleStatic(w, req, req.URL.Path)
    } else if req.URL.Path == "/" {
        html := render_html("network.html", values, true)
        w.Write([]byte(html))
     } else if req.URL.Path == "/about" {
        html := render_html("about.html", values, true)
        w.Write([]byte(html))
    }  else if req.URL.Path == "/config" {
        if req.Method == "GET" {
            values["rewriteJs"] = html.EscapeString(ser.RewriteJs)
            values["hosts"] = ""
            values["hostsHeight"] = getTextAreaHeightByString("",100)
            
            values["rewriteJsPath"] = filepath.Base(ser.GetRewriteJsPath())
            values["jsHeight"] = getTextAreaHeightByString(ser.RewriteJs,100)
            html := render_html("config.html", values, true)
            w.Write([]byte(html))
        } else if req.Method == "POST" {
            ser.handleConfig(w, req)
        }
    } else if req.URL.Path == "/response" {
        ser.showResponseById(w, req)
    } else {
        http.NotFound(w, req)
    }
}

func getTextAreaHeightByString(mystr string,minHeight int) int{
     height:=(len(strings.Split(mystr,"\n"))+1)*25
     if(height<minHeight){
        height=minHeight
     }
     return height
}

func (ser *ProxyServe) showResponseById(w http.ResponseWriter, req *http.Request) {
    id := req.FormValue("id")
    docid, uint_parse_err := strconv.ParseUint(id, 10, 64)
    if uint_parse_err == nil {
        responseData := ser.GetResponseByDocid(docid)
        if responseData == nil {
            w.Write([]byte("response not found"))
        } else {
            walker := goutils.NewInterfaceWalker(map[string]interface{}(responseData))
            content_type := ""
            if type_header, has := walker.GetStringSlice("/header/Content-Type"); has {
                content_type = strings.Join(type_header, ";")
            }

            custom_content_type := req.FormValue("type")
            //set custom content type
            if custom_content_type != "" {
                switch custom_content_type {
                case "json":
                    content_type = "application/json"
                case "html":
                    content_type = "text/html;charset=utf-8"
                default:
                    content_type = custom_content_type
                }
            }
            if content_type != "" {
                w.Header().Set("Content-Type", content_type)
            }
            if body_str, has := walker.GetString("/body"); has {
                body_byte, err := base64.StdEncoding.DecodeString(body_str)
                if err == nil {
                    w.Write(body_byte)
                } else {
                    log.Println("decode body failed", err)
                }
            } else {
                w.Write([]byte("response body not found"))
            }
        }
    } else {
        w.Write([]byte("param err"))
    }
}

func (ser *ProxyServe) handleConfig(w http.ResponseWriter, req *http.Request) {
    ser.mu.Lock()
    defer ser.mu.Unlock()
    do := req.PostFormValue("type")
    var err error
    if(do=="js"){
	    jsStr := strings.TrimSpace(req.PostFormValue("js"))
	    err= ser.parseAndSaveRewriteJs(jsStr)
	    if err == nil {
	        jsPath:=ser.GetRewriteJsPath()
	        err = goutils.File_put_contents(jsPath, []byte(jsStr))
	        log.Println("save rewritejs ", jsPath, err)
	    } 
    }else if(do=="hosts"){
	    hosts := strings.TrimSpace(req.PostFormValue("hosts"))
	    err = goutils.File_put_contents(ser.GetHostsFilePath(), []byte(hosts))
    }
    if(err!=nil) {
	  w.Write([]byte("save failed,err:" + err.Error()))
	}else{
   	   w.Write([]byte("<html>save suc<script>setTimeout(function(){location.href='/config'},1000)</script></html>"))
	}
    
}

func render_html(fileName string, values map[string]interface{}, layout bool) string {
    html := goutils.DefaultResource.Load("/res/tpl/" + fileName)
    tpl, _ := template.New("page").Parse(string(html))
    var bf []byte
    w := bytes.NewBuffer(bf)
    tpl.Execute(w, values)
    body := w.String()
    if layout {
        values["body"] = body
        values["version"] = "0.2"
        return render_html("layout.html", values, false)
    }
    return goutils.Html_reduceSpace(body)
}

func (ser *ProxyServe) handleUserInfo(w http.ResponseWriter, req *http.Request) {
	host, _, _ := net.SplitHostPort(req.RemoteAddr)
   data:="client ip:"+host
	w.Write([]byte(data))
}

package serve

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/hidu/goutils"
	"html"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"text/template"
)

var CookieName = "pproxy"

func (ser *ProxyServe) web_checkLogin(req *http.Request) (user *User, isLogin bool) {
	if req == nil {
		return
	}
	cookie, err := req.Cookie(CookieName)
	if err != nil {
		return
	}
	info := strings.SplitN(cookie.Value, ":", 2)
	if len(info) != 2 {
		return
	}
	if user, has := ser.Users[info[0]]; has {
		if user.Psw == info[1] {
			return user, true
		}
	}
	return
}

func (ser *ProxyServe) handleLocalReq(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/socket.io/1/") {
		ser.ws.ServeHTTP(w, req)
		return
	}
	values := make(map[string]interface{})
	values["title"] = ser.conf.Title
	values["subTitle"] = ""
	values["version"] = PproxyVersion
	values["notice"] = ser.conf.Notice
	values["port"] = fmt.Sprintf("%d", ser.conf.Port)
	values["userOnlineTotal"] = len(ser.wsClients) + 1

	user, isLogin := ser.web_checkLogin(req)
	values["isLogin"] = isLogin
	values["user"] = user

	if strings.HasPrefix(req.URL.Path, "/res/") {
		utils.DefaultResource.HandleStatic(w, req, req.URL.Path)
	} else if req.URL.Path == "/" {
		html := render_html("network.html", values, true)
		w.Write([]byte(html))
	} else if req.URL.Path == "/about" {
		values["subTitle"] = "about|"
		html := render_html("about.html", values, true)
		w.Write([]byte(html))
	} else if req.URL.Path == "/config" {
		values["subTitle"] = "config|"
		if req.Method == "GET" {
			values["rewriteJs"] = html.EscapeString(ser.RewriteJs)
			values["jsHeight"] = getTextAreaHeightByString(ser.RewriteJs, 100)

			hosts_byte, _ := utils.File_get_contents(ser.GetHostsFilePath())
			values["hosts"] = html.EscapeString(string(hosts_byte))
			values["hostsHeight"] = getTextAreaHeightByString("", 100)

			html := render_html("config.html", values, true)
			w.Write([]byte(html))
		} else if req.Method == "POST" {
			ser.web_handleConfig(w, req)
		}
	} else if req.URL.Path == "/login" {
		if req.Method == "GET" {
			values["subTitle"] = "login|"
			html := render_html("login.html", values, true)
			w.Write([]byte(html))
		} else {
			ser.handleLogin(w, req)
		}
	} else if req.URL.Path == "/response" {
		ser.web_showResponseById(w, req)
	} else if req.URL.Path == "/redo" {
		ser.req_redo(w, req, values)
	} else if req.URL.Path == "/logout" {
		cookie := &http.Cookie{Name: CookieName, Value: "", Path: "/"}
		http.SetCookie(w, cookie)
		http.Redirect(w, req, "/", 302)
	} else {
		http.NotFound(w, req)
	}
}
func (ser *ProxyServe) handleLogin(w http.ResponseWriter, req *http.Request) {
	name := strings.TrimSpace(req.FormValue("name"))
	psw := strings.TrimSpace(req.FormValue("psw"))
	if name == "" {
		w.Write([]byte("<script>alert('empty name!')</script>"))
		return
	}
	if user, has := ser.Users[name]; has {
		if user.isPswEq(psw) {
			log.Println("login suc,name=", name)
			cookie := &http.Cookie{Name: CookieName, Value: fmt.Sprintf("%s:%s", name, user.Psw), Path: "/"}
			http.SetCookie(w, cookie)
			w.Write([]byte("<script>parent.location.href='/'</script>"))
		} else {
			log.Println("login failed psw incorrect,name=", name, "psw=", psw)
			w.Write([]byte("<script>alert('password incorrect')</script>"))
		}
		return
	}
	log.Println("login failed not exists,name=", name, "psw=", psw)
	w.Write([]byte("<script>alert('user not exists')</script>"))
}

func (ser *ProxyServe) web_showResponseById(w http.ResponseWriter, req *http.Request) {
	id := req.FormValue("id")
	docid, uint_parse_err := strconv.ParseUint(id, 10, 64)
	if uint_parse_err == nil {
		responseData := ser.GetResponseByDocid(docid)
		if responseData == nil {
			w.Write([]byte("response not found"))
		} else {
			walker := utils.NewInterfaceWalker(map[string]interface{}(responseData))
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
			if statusCode, has := walker.GetInt("/status"); has {
				w.WriteHeader(statusCode)
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

func (ser *ProxyServe) web_handleConfig(w http.ResponseWriter, req *http.Request) {
	user, isLogin := ser.web_checkLogin(req)
	if !isLogin || !user.IsAdmin {
		w.Write([]byte("<script>alert('you are not admin')</script>"))
		return
	}
	do := req.PostFormValue("type")
	var err error
	if do == "js" {
		jsStr := strings.TrimSpace(req.PostFormValue("js"))
		err = ser.parseAndSaveRewriteJs(jsStr)
		if err == nil {
			jsPath := ser.GetRewriteJsPath()
			err = utils.File_put_contents(jsPath, []byte(jsStr))
			log.Println("save rewritejs ", jsPath, err)
		}
	} else if do == "hosts" {
		hosts := strings.TrimSpace(req.PostFormValue("hosts"))
		log.Println("hosts_update", hosts)
		err = utils.File_put_contents(ser.GetHostsFilePath(), []byte(hosts))
		ser.loadHosts()
	}
	if err != nil {
		w.Write([]byte("<script>alert('save failed,err:" + html.EscapeString(err.Error()) + ")"))
	} else {
		w.Write([]byte("<script>alert('save success');top.location.href='/config'</script>"))
	}

}

func render_html(fileName string, values map[string]interface{}, layout bool) string {
	html := utils.DefaultResource.Load("/res/tpl/" + fileName)
	tpl, _ := template.New("page").Parse(string(html))
	var bf []byte
	w := bytes.NewBuffer(bf)
	tpl.Execute(w, values)
	body := w.String()
	if layout {
		values["body"] = body
		return render_html("layout.html", values, false)
	}
	return utils.Html_reduceSpace(body)
}

func (ser *ProxyServe) handleUserInfo(w http.ResponseWriter, req *http.Request) {
	host, _, _ := net.SplitHostPort(req.RemoteAddr)
	data := "client ip:" + host
	w.Write([]byte(data))
}

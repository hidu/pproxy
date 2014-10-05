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
	"strings"
	"text/template"
	"time"
)

type webRequestCtx struct {
	values  map[string]interface{}
	user    *User
	isLogin bool
	isAdmin bool
	req     *http.Request
	w       http.ResponseWriter
	ser     *ProxyServe
}

var CookieName = "pproxy"

func (ser *ProxyServe) handleLocalReq(w http.ResponseWriter, req *http.Request) {
	accessLogStr := "web_access " + req.Method + " " + req.URL.String() + " " + req.RemoteAddr + " refer:" + req.Referer()
	defer (func() {
		log.Println(accessLogStr)
	})()

	if strings.HasPrefix(req.URL.Path, "/socket.io/") {
		ser.wsSer.server.ServeHTTP(w, req)
		return
	}

	if strings.HasPrefix(req.URL.Path, "/f/") {
		req.URL.Path = req.URL.Path[3:]
		http.FileServer(http.Dir(ser.conf.FileDir)).ServeHTTP(w, req)
		return
	}

	if strings.HasPrefix(req.URL.Path, "/res/") {
		utils.DefaultResource.HandleStatic(w, req, req.URL.Path)
		return
	}

	values := make(map[string]interface{})
	values["title"] = ser.conf.Title
	values["subTitle"] = ""
	values["version"] = PproxyVersion
	values["notice"] = ser.conf.Notice
	values["port"] = fmt.Sprintf("%d", ser.conf.Port)
	values["userOnlineTotal"] = len(ser.ProxyClients)
	_host, _port, _ := getHostPortFromReq(req)
	values["pproxy_host"] = _host
	values["pproxy_port"] = _port

	ctx := &webRequestCtx{
		values: values,
		w:      w,
		req:    req,
		ser:    ser,
	}
	ctx.checkLogin()
	values["isLogin"] = ctx.isLogin
	values["user"] = ctx.user
	if ctx.isLogin {
		values["isAdmin"] = ctx.user.IsAdmin
	} else {
		values["isAdmin"] = false
	}

	funcMap := make(map[string]func())
	funcMap["/"] = ctx.handle_index
	funcMap["/about"] = ctx.handle_about
	funcMap["/config"] = ctx.handle_config
	funcMap["/useage"] = ctx.handle_useage
	funcMap["/replay"] = ctx.handle_replay
	funcMap["/login"] = ctx.handle_login
	funcMap["/logout"] = ctx.handle_logout
	funcMap["/response"] = ctx.handle_response

	if fn, has := funcMap[req.URL.Path]; has {
		if len(req.URL.Path) > 1 {
			ctx.values["subTitle"] = req.URL.Path[1:] + " |"
		}
		fn()
	} else {
		http.NotFound(w, req)
	}
}

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
		if user.PswMd5 == info[1] {
			return user, true
		}
	}
	return
}

func (ctx *webRequestCtx) checkLogin() {
	user, isLogin := ctx.ser.web_checkLogin(ctx.req)
	if isLogin {
		ctx.user = user
		ctx.isLogin = true
		ctx.isAdmin = user.IsAdmin
	}
}

func (ctx *webRequestCtx) render(name string, layout bool) {
	html := render_html(name, ctx.values, layout)
	ctx.w.Write([]byte(html))
}

func (ctx *webRequestCtx) handle_index() {
	ctx.render("network.html", true)
}

func (ctx *webRequestCtx) handle_useage() {
	ctx.render("useage.html", true)
}
func (ctx *webRequestCtx) handle_config() {
	if ctx.req.Method == "GET" {
		_jsDataArr := make([]interface{}, 0, 2)
		jsDefault := make(map[string]interface{})

		jsStr, _ := ctx.ser.reqMod.getJsContent("")
		jsDefault["title"] = "global config"
		jsDefault["name"] = ""
		jsDefault["rewriteJs"] = html.EscapeString(jsStr)
		jsDefault["jsHeight"] = getTextAreaHeightByString(jsStr, 100)
		_jsDataArr = append(_jsDataArr, jsDefault)

		if ctx.isLogin {
			jsUser := make(map[string]interface{})
			jsStr, _ := ctx.ser.reqMod.getJsContent(ctx.user.Name)
			jsUser["title"] = fmt.Sprintf("user's config-[%s]", ctx.user.Name)
			jsUser["name"] = ctx.user.Name
			jsUser["rewriteJs"] = html.EscapeString(jsStr)
			jsUser["jsHeight"] = getTextAreaHeightByString(jsStr, 100)
			_jsDataArr = append(_jsDataArr, jsUser)
		}

		ctx.values["jss"] = _jsDataArr

		hosts_byte, _ := utils.File_get_contents(ctx.ser.GetHostsFilePath())
		ctx.values["hosts"] = html.EscapeString(string(hosts_byte))
		ctx.values["hostsHeight"] = getTextAreaHeightByString("", 100)

		ctx.render("config.html", true)
	} else if ctx.req.Method == "POST" {
		if !ctx.isLogin {
			ctx.jsAlert("login first")
			return
		}
		do := ctx.req.PostFormValue("type")
		var err error
		if do == "js" {
			name := strings.TrimSpace(ctx.req.PostFormValue("name"))
			if !ctx.isAdmin && name != ctx.user.Name {
				ctx.jsAlert("you are not admin")
				return
			}
			jsStr := strings.TrimSpace(ctx.req.PostFormValue("js"))
			err = ctx.ser.reqMod.parseJs(jsStr, name, true)
		} else if do == "hosts" {
			if !ctx.isAdmin {
				ctx.jsAlert("you are not admin")
				return
			}
			hosts := strings.TrimSpace(ctx.req.PostFormValue("hosts"))
			log.Println("hosts_update", hosts)
			err = utils.File_put_contents(ctx.ser.GetHostsFilePath(), []byte(hosts))
			ctx.ser.loadHosts()
		}
		if err != nil {
			ctx.jsAlert("save failed,err:" + err.Error())
		} else {
			ctx.w.Write([]byte("<script>alert('save success');top.location.href='/config'</script>"))
		}
	}

}
func (ctx *webRequestCtx) handle_response() {
	docid, uint_parse_err := parseDocId(ctx.req.FormValue("id"))
	if uint_parse_err == nil {
		responseData := ctx.ser.GetResponseByDocid(docid)
		if responseData == nil {
			ctx.w.Write([]byte("response not found"))
		} else {
			walker := utils.NewInterfaceWalker(map[string]interface{}(responseData))
			content_type := ""
			if type_header, has := walker.GetStringSlice("/header/Content-Type"); has {
				content_type = strings.Join(type_header, ";")
			}

			custom_content_type := ctx.req.FormValue("type")
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
				ctx.w.Header().Set("Content-Type", content_type)
			}
			if statusCode, has := walker.GetInt("/status"); has {
				ctx.w.WriteHeader(statusCode)
			}
			if body_str, has := walker.GetString("/body"); has {
				body_byte, err := base64.StdEncoding.DecodeString(body_str)
				if err == nil {
					ctx.w.Write(body_byte)
				} else {
					log.Println("decode body failed", err)
				}
			} else {
				ctx.w.Write([]byte("response body not found"))
			}
		}
	} else {
		ctx.w.Write([]byte("param err"))
	}
}

func (ctx *webRequestCtx) jsAlert(msg string) {
	ctx.w.Write([]byte(fmt.Sprintf("<script>alert('%s')</script>", html.EscapeString(msg))))
}

func (ctx *webRequestCtx) handle_about() {
	ctx.render("about.html", true)
}

func (ctx *webRequestCtx) handle_logout() {
	cookie := &http.Cookie{Name: CookieName, Value: "", Path: "/"}
	http.SetCookie(ctx.w, cookie)
	http.Redirect(ctx.w, ctx.req, "/", 302)
}

func (ctx *webRequestCtx) handle_login() {
	if ctx.req.Method == "GET" {
		ctx.render("login.html", true)
	} else {
		name := strings.TrimSpace(ctx.req.FormValue("name"))
		psw := strings.TrimSpace(ctx.req.FormValue("psw"))
		if name == "" {
			ctx.jsAlert("empty name")
			return
		}
		if user, has := ctx.ser.Users[name]; has {
			if user.isPswEq(psw) {
				log.Println("login suc,name=", name)
				cookie := &http.Cookie{
					Name:    CookieName,
					Value:   fmt.Sprintf("%s:%s", name, user.PswMd5),
					Path:    "/",
					Expires: time.Now().Add(86400 * time.Second),
				}
				http.SetCookie(ctx.w, cookie)
				ctx.w.Write([]byte("<script>parent.location.href='/'</script>"))
			} else {
				log.Println("login failed psw incorrect,name=", name, "psw=", psw)
				ctx.jsAlert("password incorrect")
			}
			return
		}
		log.Println("login failed not exists,name=", name, "psw=", psw)
		ctx.jsAlert("user not exists")
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

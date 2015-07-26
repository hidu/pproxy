package serve

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	REPLAY_FLAG       = "Proxy-pproxy_replay"
	REPLAY_REMOTEADDR = "Proxy-pproxy_remoteaddr"
	REPLAY_USER_NAME  = "Proxy-pproxy_user"
)

func (ctx *webRequestCtx) handle_replay() {
	if ctx.req.Method == "POST" {
		ctx.req_replayPost()
		return
	}
	docid_str := strings.TrimSpace(ctx.req.FormValue("id"))
	if docid_str == "" {
		ctx.w.WriteHeader(http.StatusBadRequest)
		ctx.w.Write([]byte("empty id param"))
		return
	}
	docid, err_int := parseDocId(docid_str)
	if err_int != nil {
		ctx.w.WriteHeader(http.StatusInternalServerError)
		ctx.w.Write([]byte(fmt.Sprintf("param id[%s] error:\n%s", docid_str, err_int)))
		return
	}
	req_doc, _ := ctx.ser.GetRequestByDocid(docid)
	if req_doc == nil {
		ctx.w.WriteHeader(http.StatusNotFound)
		ctx.w.Write([]byte("request doc not found!"))
		return
	}
	_url := fmt.Sprintf("%s", req_doc.Data["url"])
	u, err := url.Parse(_url)
	if err != nil {
		ctx.w.WriteHeader(http.StatusInternalServerError)
		ctx.w.Write([]byte(fmt.Sprintf("parse url[%s] error\n%s", _url, err)))
		return
	}
	u.RawQuery = ""

	ctx.values["req"] = req_doc
	ctx.values["action_url"] = u.String()

	ctx.render("replay.html", true)
}

var replay_skip_headers = map[string]int{"Content-Length": 1}

func (ctx *webRequestCtx) req_replayPost() {
	replay := ctx.req.FormValue("replay")
	basic := make(map[string]string)
	basic["action_url"] = strings.TrimSpace(ctx.req.FormValue("basic_action_url"))
	method := strings.TrimSpace(strings.ToUpper(ctx.req.FormValue("basic_method")))
	basic["method"] = method

	host := strings.TrimSpace(ctx.req.FormValue("basic_host"))

	basic_remoteAddr := ctx.req.FormValue("basic_RemoteAddr")
	basic_user := ctx.req.FormValue("basic_user")

	header := GetFormValuesWithPrefix(ctx.req.Form, "header_")
	get := GetFormValuesWithPrefix(ctx.req.Form, "get_")
	post := GetFormValuesWithPrefix(ctx.req.Form, "post_")

	formData := make(map[string]interface{})
	formData["basic"] = basic

	formData["header"] = header
	formData["get"] = get
	formData["post"] = post

	ctx.values["form"] = formData
	if replay == "direct" {
		ctx.render("replay_direct.html", true)
		return
	} else {
		req_bd := ""
		_url := basic["action_url"]

		if len(get) > 0 {
			form_values := make(url.Values)
			for k, v := range get {
				for _, _v := range v {
					form_values.Add(k, _v)
				}
			}
			if strings.Contains(_url, "?") {
				_url += "&"
			} else {
				_url += "?"
			}
			_url += form_values.Encode()
		}

		if len(post) > 0 {
			form_values := make(url.Values)
			for k, v := range post {
				for _, _v := range v {
					form_values.Add(k, _v)
				}
			}
			req_bd = form_values.Encode()
		}

		replay_req, err := http.NewRequest(method, _url, strings.NewReader(req_bd))
		if err != nil {
			ctx.w.Write([]byte("build request failed\n" + err.Error()))
			return
		}
		if host != "" {
			replay_req.Host = host
		}
		replay_req.Header.Set(REPLAY_FLAG, "replay")

		replay_req.Header.Set(REPLAY_REMOTEADDR, basic_remoteAddr)
		replay_req.Header.Set(REPLAY_USER_NAME, basic_user)

		for k, v := range header {
			if _, has := replay_skip_headers[k]; has {
				continue
			}
			replay_req.Header.Set(k, strings.Join(v, ";"))
		}
		ctx.ser.ServeHTTPProxy(ctx.w, replay_req)
	}
}

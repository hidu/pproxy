package serve

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	REDO_FLAG       = "Proxy-pproxy_redo"
	REDO_REMOTEADDR = "Proxy-pproxy_remoteaddr"
	REDO_USER_NAME  = "Proxy-pproxy_user"
)

func (ser *ProxyServe) req_redo(w http.ResponseWriter, req *http.Request, values map[string]interface{}) {
	if req.Method == "POST" {
		ser.req_redoPost(w, req, values)
		return
	}
	docid_str := strings.TrimSpace(req.FormValue("id"))
	if docid_str == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("empty id param"))
		return
	}
	docid, err_int := strconv.ParseUint(docid_str, 10, 64)
	if err_int != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("param id[%s] error:\n%s", docid_str, err_int)))
		return
	}
	req_doc := ser.GetRequestByDocid(docid)
	if req_doc == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("request doc not found!"))
		return
	}
	_url := fmt.Sprintf("%s", req_doc["url"])
	u, err := url.Parse(_url)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("parse url[%s] error\n%s", _url, err)))
		return
	}
	u.RawQuery = ""
	values["req"] = req_doc
	values["action_url"] = u.String()
	values["subTitle"] = "redo|" + u.String() + "|"
	html := render_html("redo.html", values, true)
	w.Write([]byte(html))
}

var redo_skip_headers = map[string]int{"Content-Length": 1}

func (ser *ProxyServe) req_redoPost(w http.ResponseWriter, req *http.Request, values map[string]interface{}) {
	redo := req.FormValue("redo")
	basic := make(map[string]string)
	basic["action_url"] = strings.TrimSpace(req.FormValue("basic_action_url"))
	method := strings.TrimSpace(strings.ToUpper(req.FormValue("basic_method")))
	basic["method"] = method

	basic_remoteAddr := req.FormValue("basic_RemoteAddr")
	basic_user := req.FormValue("basic_user")

	header := GetFormValuesWithPrefix(req.Form, "header_")
	get := GetFormValuesWithPrefix(req.Form, "get_")
	post := GetFormValuesWithPrefix(req.Form, "post_")

	formData := make(map[string]interface{})
	formData["basic"] = basic

	formData["header"] = header
	formData["get"] = get
	formData["post"] = post

	values["form"] = formData
	if redo == "direct" {
		html := render_html("redo_direct.html", values, true)
		w.Write([]byte(html))
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
			if(strings.Contains(_url,"?")){
			 _url+="&"
			}else{
			 _url+="?"
			}
			_url +=form_values.Encode()
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

		redo_req, err := http.NewRequest(method, _url, strings.NewReader(req_bd))
		if err != nil {
			w.Write([]byte("build request failed\n" + err.Error()))
			return
		}

		redo_req.Header.Set(REDO_FLAG, "redo")

		redo_req.Header.Set(REDO_REMOTEADDR, basic_remoteAddr)
		redo_req.Header.Set(REDO_USER_NAME, basic_user)

		for k, v := range header {
			if _, has := redo_skip_headers[k]; has {
				continue
			}
			redo_req.Header.Set(k, strings.Join(v, ";"))
		}
		ser.Goproxy.ServeHTTP(w, redo_req)
	}
}

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

func (ctx *webRequestCtx) handleReplay() {
	if ctx.req.Method == "POST" {
		ctx.reqReplayPost()
		return
	}
	docidStr := strings.TrimSpace(ctx.req.FormValue("id"))
	if docidStr == "" {
		ctx.w.WriteHeader(http.StatusBadRequest)
		ctx.w.Write([]byte("empty id param"))
		return
	}
	docid, errInt := parseDocID(docidStr)
	if errInt != nil {
		ctx.w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(ctx.w, "param id[%s] error:\n%s", docidStr, errInt)
		return
	}
	reqDoc, _ := ctx.ser.getRequestByDocid(docid)
	if reqDoc == nil {
		ctx.w.WriteHeader(http.StatusNotFound)
		ctx.w.Write([]byte("request doc not found!"))
		return
	}
	_url := fmt.Sprintf("%s", reqDoc.Data["url"])
	u, err := url.Parse(_url)
	if err != nil {
		ctx.w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(ctx.w, "parse url[%s] error\n%s", _url, err)
		return
	}
	u.RawQuery = ""

	ctx.values["req"] = reqDoc
	ctx.values["action_url"] = u.String()

	ctx.render("replay.html", true)
}

var replaySkipHeaders = map[string]int{"Content-Length": 1}

func (ctx *webRequestCtx) reqReplayPost() {
	replay := ctx.req.FormValue("replay")
	basic := make(map[string]string)
	basic["action_url"] = strings.TrimSpace(ctx.req.FormValue("basic_action_url"))
	method := strings.TrimSpace(strings.ToUpper(ctx.req.FormValue("basic_method")))
	basic["method"] = method

	host := strings.TrimSpace(ctx.req.FormValue("basic_host"))

	basicRemoteAddr := ctx.req.FormValue("basic_RemoteAddr")
	basicUser := ctx.req.FormValue("basic_user")

	header := getFormValuesWithPrefix(ctx.req.Form, "header_")
	get := getFormValuesWithPrefix(ctx.req.Form, "get_")
	post := getFormValuesWithPrefix(ctx.req.Form, "post_")

	formData := make(map[string]any)
	formData["basic"] = basic

	formData["header"] = header
	formData["get"] = get
	formData["post"] = post

	ctx.values["form"] = formData
	if replay == "direct" {
		ctx.render("replay_direct.html", true)
		return
	}
	reqBd := ""
	_url := basic["action_url"]

	if len(get) > 0 {
		formValues := make(url.Values)
		for k, v := range get {
			for _, _v := range v {
				formValues.Add(k, _v)
			}
		}
		if strings.Contains(_url, "?") {
			_url += "&"
		} else {
			_url += "?"
		}
		_url += formValues.Encode()
	}

	if len(post) > 0 {
		formValues := make(url.Values)
		for k, v := range post {
			for _, _v := range v {
				formValues.Add(k, _v)
			}
		}
		reqBd = formValues.Encode()
	}

	replayReq, err := http.NewRequest(method, _url, strings.NewReader(reqBd))
	if err != nil {
		ctx.w.Write([]byte("build request failed\n" + err.Error()))
		return
	}
	if host != "" {
		replayReq.Host = host
	}
	replayReq.Header.Set(REPLAY_FLAG, "replay")

	replayReq.Header.Set(REPLAY_REMOTEADDR, basicRemoteAddr)
	replayReq.Header.Set(REPLAY_USER_NAME, basicUser)

	for k, v := range header {
		if _, has := replaySkipHeaders[k]; has {
			continue
		}
		replayReq.Header.Set(k, strings.Join(v, ";"))
	}
	ctx.ser.ServeHTTPProxy(ctx.w, replayReq)
}

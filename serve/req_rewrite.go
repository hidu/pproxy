package serve

import (
	"bytes"
	"fmt"
	"github.com/hidu/goutils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

//var rewriteJsTpl = "function pproxy_rewrite(req){\n%s\nreturn req;\n}"
var rewriteJsTpl = string(utils.DefaultResource.Load("res/sjs/req_rewrite.js"))

func (ser *ProxyServe) parseAndSaveRewriteJs(jsStr string) error {
	rewriteJs := strings.Replace(rewriteJsTpl, "CUSTOM_JS", jsStr, 1)
	js.Run(rewriteJs)
	jsFn, err := js.Get("pproxy_rewrite")
	if err == nil {
		ser.RewriteJs = jsStr
		ser.RewriteJsFn = jsFn
	} else {
		log.Println("rewrite js init error:", err)
	}
	return err
}

func (ser *ProxyServe) reqRewriteByjs(req *http.Request, reqCtx *requestCtx) int {
	if ser.RewriteJs == "" {
		return 304
	}
	schema := req.URL.Scheme
	origin_url := req.URL.String()
	origin_get_query := req.URL.Query()
	///================================================================
	headerKv := make(map[string]string)
	headerKv["method"] = req.Method
	headerKv["schema"] = schema
	headerKv["path"] = req.URL.Path

	_host, port_int, _ := getHostPortFromReq(req)

	headerKv["host"] = _host
	headerKv["port"] = fmt.Sprintf("%d", port_int)

	username := ""
	psw := ""
	if req.URL.User != nil {
		username = req.URL.User.Username()
		psw, _ = req.URL.User.Password()
	}
	headerKv["username"] = username
	
	headerKv["proxy_user"] = reqCtx.User.Name
	
	headerKv["password"] = psw

	//===================================================================
	rewriteData := make(map[string]interface{})
	rewriteData["header"] = headerKv
	rewriteData["get"] = origin_get_query
	rewriteData["post"] = *reqCtx.FormPost

	reqJsObj, _ := js.Object(`req={}`)
	reqJsObj.Set("origin", rewriteData)

	///===================================

	js_ret, err_js := ser.RewriteJsFn.Call(ser.RewriteJsFn, reqJsObj)

	if err_js != nil {
		log.Println("js filter err:", err_js, js_ret)
		return 502
	}
	if !js_ret.IsObject() {
		log.Println("wrong req_rewirte return value")
		return 502
	}
	obj, export_err := js_ret.Export()

	if export_err != nil {
		log.Println("js filter result wrong", js_ret.String())
		return 502
	}
	//================================================================================

	reqObjNew := obj.(map[string]interface{})

	headerKvNew := make(map[string]string)
	isHeaderChange := false

	for k, v := range headerKv {
		_newVal := getMapValStr(reqObjNew, k)
		headerKvNew[k] = _newVal
		if _newVal != v {
			isHeaderChange = true
		}
	}
	//-------------------------------------------------------
	var get_new url.Values

	isGetChange := false

	if _get, has := reqObjNew["get"]; has {
		get_new = _req_mapToUrlValue(_get)
		isGetChange = checkUrlValuesChange(origin_get_query, get_new)
	}
	//-------------------------------------------------------
	var post_new url.Values
	isPostChange := false
	if schema == "http" {
		if _post, has := reqObjNew["post"]; has {
			post_new = _req_mapToUrlValue(_post)
			isPostChange = checkUrlValuesChange(*reqCtx.FormPost, post_new)
		}
	}

	host_addr := getMapValStr(reqObjNew, "host_addr")
	isHostAddrChange := host_addr != ""

	if ser.Debug {
		fmt.Println("rewriteChange:", "is_get_change:", isGetChange, "new_get:", get_new,
			"isPostChange:", isPostChange, "new_post:", post_new,
			"isHostAddrChange:", isHostAddrChange, "new_host_addr:", host_addr,
		)
	}

	///===============================================================================
	if !isHeaderChange && !isGetChange && !isPostChange && !isHostAddrChange {
		return 304
	}
	///===============================================================================

	var url_base string

	if isHeaderChange {
		//		schema := headerKvNew["schema"]
		url_base = schema + "://"

		if headerKvNew["username"] != "" {
			url_base += fmt.Sprintf("%s:%s@", headerKvNew["username"], headerKvNew["password"])
		}
		url_base += headerKvNew["host"]
		if headerKvNew["port"] != "" && headerKvNew["port"] != "80" {
			url_base += ":" + headerKvNew["port"]
		}
		url_base += headerKvNew["path"]
	} else {
		if req.URL.RawQuery == "" {
			url_base = origin_url
		} else {
			url_base = origin_url[:len(origin_url)-len(req.URL.RawQuery)-1]
		}
	}

	if isGetChange {
		url_base += "?" + get_new.Encode()
	} else {
		url_base += "?" + req.URL.RawQuery
	}

	if isHeaderChange || isGetChange {
		var url_err error
		req.URL, url_err = url.Parse(url_base)
		if ser.Debug {
			log.Println("DEBUG req_rewrite,url_new:", url_base, "req_new:", req.URL)
		}
		if url_err != nil {
			log.Println("js filter err:", js_ret, url_err)
			return 502
		}

		req.Host = req.URL.Host
	}

	//////////////////////////////////////////////////////////////////////////////

	if isPostChange {
		buf := bytes.NewBuffer([]byte{})
		_post_body := post_new.Encode()
		req.Header.Del("Content-Length")
		if req.Header.Get(Content_Encoding) == "gzip" {
			tmp := gzipEncode([]byte(_post_body)).Bytes()
			buf.Write(tmp)
		} else {
			buf.WriteString(_post_body)
		}
		req.ContentLength = int64(buf.Len())
		req.Body = ioutil.NopCloser(buf).(io.ReadCloser)
	}

	////////////////////////////////////////////////////////////////////////////

	if isHostAddrChange {
		req.URL.Host = host_addr
		if ser.Debug {
			log.Println("rewrite host addr:", req.URL.Host, "==>", host_addr)
		}
	}
	return 200
}

func (ser *ProxyServe) reqRewrite(req *http.Request, reqCtx *requestCtx) int {
	origin_host := req.Host + "#" + req.URL.Host
	statusCode1 := ser.reqRewriteByjs(req, reqCtx)
	new_host := req.Host + "#" + req.URL.Host

	if ser.Debug {
		log.Println("rewrte_debug:\n", "origin_host:", origin_host, "\nnew_host:", new_host, "\n")
	}

	statusCode2 := 304
	if origin_host == new_host {
		statusCode2 = ser.reqRewriteByHosts(req)
	}
	if statusCode1 == 200 || statusCode2 == 200 {
		return 200
	}
	if statusCode1 >= 500 || statusCode2 >= 500 {
		return 502
	}
	return 304
}

func (ser *ProxyServe) reqRewriteByHosts(req *http.Request) int {
	if ser.hosts == nil {
		return 304
	}
	if host, has := ser.hosts[req.URL.Host]; has {
		log.Println("rewrite host:", req.URL.Host, "==>", host)
		req.URL.Host = host
		return 200
	}
	host_info := strings.Split(req.URL.Host, ":")
	if len(host_info) == 1 {
		if req.URL.Scheme == "http" {
			host_info = append(host_info, "80")
		}
	}
	req_host := strings.Join(host_info, ":")
	if host, has := ser.hosts[req_host]; has {
		log.Println("rewrite host:", req.Host, "==>", host)
		req.URL.Host = host
		return 200
	}

	if host, has := ser.hosts[host_info[0]]; has {
		log.Println("rewrite host:", req.Host, "==>", host)
		req.URL.Host = host
		if !strings.Contains(host, ":") {
			req.URL.Host += ":" + host_info[1]
		}
		return 200
	}
	return 304
}

/**
*
 */
func _req_mapToUrlValue(values interface{}) url.Values {
	uValues := make(url.Values)
	if values == nil {
		return uValues
	}
	vs := values.(map[string]interface{})

	for k, arr := range vs {
		switch value := arr.(type) {
		case []interface{}:
			for _, v := range value {
				uValues.Add(k, fmt.Sprintf("%v", v))
			}
		case interface{}:
			uValues.Set(k, fmt.Sprintf("%v", value))
		default:
			log.Println("unkonw type:", value)
		}
	}
	return uValues
}

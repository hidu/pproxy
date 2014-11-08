package serve

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func (ser *ProxyServe) reqRewriteByjs( reqCtx *requestCtx) int {
	if !ser.reqMod.CanMod() {
		return 304
	}
	req:=reqCtx.Req
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

	reqObjNew, rErr := ser.reqMod.rewrite(rewriteData, reqCtx.User.Name)
	if rErr != nil {
		log.Println("rewrite failed:", rErr)
	}

	headerKvNew := make(map[string]string)
	isHeaderChange := false

	skipHeader := false
	var err error

	urlStrNew := getMapValStr(reqObjNew, "url")
	if urlStrNew != "" {
		req.URL, err = url.Parse(urlStrNew)
		if err != nil || req.URL.Scheme != "http" {
			log.Println("new url wrong!url is:", urlStrNew, err)
			return 500
		}
		req.Host = req.URL.Host
		skipHeader = true
	}
	if !skipHeader {
		for k, v := range headerKv {
			_newVal := getMapValStr(reqObjNew, k)
			headerKvNew[k] = _newVal
			if _newVal != v {
				isHeaderChange = true
			}
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

func (ser *ProxyServe) reqRewrite(reqCtx *requestCtx) int {
	origin_host := reqCtx.Req.Host + "#" + reqCtx.Req.URL.Host
	statusCode1 := ser.reqRewriteByjs(reqCtx)
	new_host := reqCtx.Req.Host + "#" + reqCtx.Req.URL.Host

	if ser.Debug {
		log.Println("rewrte_debug:\n", "origin_host:", origin_host, "\nnew_host:", new_host, "\n")
	}

	statusCode2 := 304
	if origin_host == new_host {
		statusCode2 = ser.reqRewriteByHosts(reqCtx.Req)
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

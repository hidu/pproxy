package serve

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (ser *ProxyServe) reqRewriteByjs(reqCtx *requestCtx) int {
	modifer := ser.reqMod
	if !modifer.CanMod() {
		return 304
	}
	req := reqCtx.Req
	schema := req.URL.Scheme
	originURL := req.URL.String()
	originGetQuery := req.URL.Query()
	// /================================================================
	headerKv := make(map[string]string)
	headerKv["method"] = req.Method
	headerKv["schema"] = schema
	headerKv["path"] = req.URL.Path

	_host, portInt, _ := getHostPortFromReq(req)

	headerKv["host"] = _host
	headerKv["port"] = strconv.Itoa(portInt)

	username := ""
	psw := ""
	if req.URL.User != nil {
		username = req.URL.User.Username()
		psw, _ = req.URL.User.Password()
	}
	headerKv["username"] = username

	headerKv["proxy_user"] = reqCtx.User.Name

	headerKv["password"] = psw

	// ===================================================================
	rewriteData := make(map[string]any)
	rewriteData["header"] = headerKv
	rewriteData["get"] = originGetQuery
	rewriteData["post"] = *reqCtx.FormPost

	_buf := forgetRead(&reqCtx.Req.Body)
	var rawBody string
	//  暂时只考虑gip的，其他的压缩就不支持了
	if req.Header.Get(contentEncoding) == "gzip" {
		rawBody = gzipDocode(_buf)
	} else {
		rawBody = _buf.String()
	}
	rewriteData["body"] = rawBody

	reqObjNew, rErr := modifer.rewrite(rewriteData, reqCtx.User.Name)
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
	// -------------------------------------------------------
	var getNew url.Values

	isGetChange := false

	if _get, has := reqObjNew["get"]; has {
		getNew = _reqMapToURLValue(_get)
		isGetChange = checkURLValuesChange(originGetQuery, getNew)
	}
	// -------------------------------------------------------
	var postNew url.Values
	isPostChange := false

	if schema == "http" {
		if _post, has := reqObjNew["post"]; has {
			postNew = _reqMapToURLValue(_post)
			isPostChange = checkURLValuesChange(*reqCtx.FormPost, postNew)
		}
	}
	isBodyChange := false
	bodyNew := ""
	if _bodyNew, has := reqObjNew["body"]; has {
		bodyNew = _bodyNew.(string)
		isBodyChange = rawBody != bodyNew
	}

	hostAddr := getMapValStr(reqObjNew, "hostAddr")
	isHostAddrChange := hostAddr != ""

	if ser.Debug {
		fmt.Println("rewriteChange:", "is_get_change:", isGetChange, "new_get:", getNew,
			"isPostChange:", isPostChange, "new_post:", postNew,
			"isHostAddrChange:", isHostAddrChange, "newHostAddr:", hostAddr,
			"isBodyChange:", isBodyChange,
		)
	}

	// /===============================================================================
	if !isHeaderChange && !isGetChange && !isPostChange && !isHostAddrChange && !isBodyChange {
		return 304
	}
	// /===============================================================================

	var urlBase string

	if isHeaderChange {
		// 		schema := headerKvNew["schema"]
		urlBase = schema + "://"

		if headerKvNew["username"] != "" {
			urlBase += fmt.Sprintf("%s:%s@", headerKvNew["username"], headerKvNew["password"])
		}
		urlBase += headerKvNew["host"]
		if headerKvNew["port"] != "" && headerKvNew["port"] != "80" {
			urlBase += ":" + headerKvNew["port"]
		}
		urlBase += headerKvNew["path"]
	} else {
		if req.URL.RawQuery == "" {
			urlBase = originURL
		} else {
			urlBase = originURL[:len(originURL)-len(req.URL.RawQuery)-1]
		}
	}

	if isGetChange {
		urlBase += "?" + getNew.Encode()
	} else {
		urlBase += "?" + req.URL.RawQuery
	}

	if isHeaderChange || isGetChange {
		var urlErr error
		req.URL, urlErr = url.Parse(urlBase)
		if ser.Debug {
			log.Println("DEBUG req_rewrite,url_new:", urlBase, "req_new:", req.URL)
		}
		if urlErr != nil {
			return 502
		}

		req.Host = req.URL.Host
	}

	// ////////////////////////////////////////////////////////////////////////////

	if isPostChange || isBodyChange {
		buf := bytes.NewBuffer([]byte{})
		var bodyData string
		if isPostChange {
			bodyData = postNew.Encode()
		} else if isBodyChange {
			bodyData = bodyNew
		}
		req.Header.Del("Content-Length")
		if req.Header.Get(contentEncoding) == "gzip" {
			tmp := gzipEncode([]byte(bodyData)).Bytes()
			buf.Write(tmp)
		} else {
			buf.WriteString(bodyData)
		}
		req.ContentLength = int64(buf.Len())
		req.Body = io.NopCloser(buf).(io.ReadCloser)
	}

	// //////////////////////////////////////////////////////////////////////////

	if isHostAddrChange {
		req.URL.Host = hostAddr
		if ser.Debug {
			log.Println("rewrite host addr:", req.URL.Host, "==>", hostAddr)
		}
	}
	return 200
}

func (ser *ProxyServe) reqRewrite(reqCtx *requestCtx) int {
	if !ser.conf.ModifyRequest {
		return 304
	}
	if reqCtx.Req.Method == "CONNECT" {
		return 304
	}
	originHost := reqCtx.Req.Host + "#" + reqCtx.Req.URL.Host
	statusCode1 := ser.reqRewriteByjs(reqCtx)
	newHost := reqCtx.Req.Host + "#" + reqCtx.Req.URL.Host

	if ser.Debug {
		log.Println("rewrte_debug:\n", "originHost:", originHost, "\nnewHost:", newHost, "\n")
	}

	statusCode2 := 304
	if originHost == newHost {
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
	hostInfo := strings.Split(req.URL.Host, ":")
	if len(hostInfo) == 1 {
		if req.URL.Scheme == "http" {
			hostInfo = append(hostInfo, "80")
		}
	}
	reqHost := strings.Join(hostInfo, ":")
	if host, has := ser.hosts[reqHost]; has {
		log.Println("rewrite host:", req.Host, "==>", host)
		req.URL.Host = host
		return 200
	}

	if host, has := ser.hosts[hostInfo[0]]; has {
		log.Println("rewrite host:", req.Host, "==>", host)
		req.URL.Host = host
		if !strings.Contains(host, ":") {
			req.URL.Host += ":" + hostInfo[1]
		}
		return 200
	}
	return 304
}

func _reqMapToURLValue(values any) url.Values {
	uValues := make(url.Values)
	if values == nil {
		return uValues
	}
	vs := values.(map[string]any)

	for k, arr := range vs {
		switch value := arr.(type) {
		case []any:
			for _, v := range value {
				uValues.Add(k, fmt.Sprintf("%v", v))
			}
		case any:
			uValues.Set(k, fmt.Sprintf("%v", value))
		default:
			log.Println("unkonw type:", value)
		}
	}
	return uValues
}

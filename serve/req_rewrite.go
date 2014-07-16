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
		return 200
	}
	urlObj, _ := js.Object(`req={}`)
	urlObj.Set("method", req.Method)
	urlObj.Set("url", req.URL.String())
	urlObj.Set("schema", req.URL.Scheme)

	host_info := strings.Split(req.URL.Host, ":")
	urlObj.Set("host", host_info[0])
	if len(host_info) == 2 {
		urlObj.Set("port", host_info[1])
	} else {
		urlObj.Set("port", "")
	}
	urlObj.Set("path", req.URL.Path)
	urlObj.Set("_pproxy_get", req.URL.Query())
	urlObj.Set("_pproxy_post", *reqCtx.FormPost)

	u_s, _ := urlObj.Value().ToString()
	fmt.Println("urlObj:", u_s)
	username := ""
	psw := ""
	if req.URL.User != nil {
		username = req.URL.User.Username()
		psw, _ = req.URL.User.Password()
	}
	urlObj.Set("username", username)
	urlObj.Set("password", psw)

	js_ret, err_js := ser.RewriteJsFn.Call(ser.RewriteJsFn, urlObj)

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
	url_obj := obj.(map[string]interface{})
	schema := getMapValStr(url_obj, "schema")
	url_new := schema + "://"
	uname := getMapValStr(url_obj, "username")
	if uname != "" {
		url_new += fmt.Sprintf("%s:%s@", uname, getMapValStr(url_obj, "password"))
	}
	host := getMapValStr(url_obj, "host")
	port := getMapValStr(url_obj, "port")
	if port != "" {
		host += ":" + port
	}
	url_new += fmt.Sprintf("%s%s", host, getMapValStr(url_obj, "path"))


	changeFlag := make(map[string]string)
	for k, v := range url_obj["flag"].(map[string]interface{}) {
		changeFlag[k] = fmt.Sprintf("%v", v)
	}
	if(ser.Debug){
		fmt.Println("changeFlag:",changeFlag)
	}

	if changeFlag["get"] == "1" {
		if _get, has := url_obj["get"]; has {
			get := _req_mapToUrlValue(_get)
			url_new += "?" + get.Encode()
			if(ser.Debug){
			   fmt.Println("rewrite_form_get:",get)
			}
		}
	}
	
	
	var url_err error
	req.URL, url_err = url.Parse(url_new)
	if ser.Debug {
		log.Println("DEBUG req_rewrite,url_new:", url_new, "req_new:", req.URL)
	}
	if url_err != nil {
		log.Println("js filter err:", js_ret, url_err)
		return 502
	}

	req.Host = req.URL.Host
	
   //////////////////////////////////////////////////////////////////////////////
	
	
	if changeFlag["post"] == "1" {
		if _post, has := url_obj["post"]; has {
			post := _req_mapToUrlValue(_post)
			buf := bytes.NewBuffer([]byte{})
			_post_body := post.Encode()
			req.Header.Del("Content-Length")
			if req.Header.Get(Content_Encoding) == "gzip" {
				tmp := gzipEncode([]byte(_post_body)).Bytes()
				buf.Write(tmp)
			} else {
				buf.WriteString(_post_body)
			}
			req.ContentLength = int64(buf.Len())
			req.Body = ioutil.NopCloser(buf).(io.ReadCloser)
			
			if(ser.Debug){
			   fmt.Println("rewrite_form_post:",post)
			}
		}
	}
	////////////////////////////////////////////////////////////////////////////
	host_addr := getMapValStr(url_obj, "host_addr")
	if host_addr != "" {
		req.URL.Host = host_addr
	}
	return 200
}

func (ser *ProxyServe) reqRewrite(req *http.Request, reqCtx *requestCtx) int{
	ret:=ser.reqRewriteByjs(req, reqCtx)
	ser.reqRewriteByHosts(req)
	return ret
}

func (ser *ProxyServe) reqRewriteByHosts(req *http.Request) {
	if ser.hosts == nil {
		return
	}
	if host, has := ser.hosts[req.URL.Host]; has {
		log.Println("rewrite host:", req.URL.Host, "==>", host)
		req.URL.Host = host
		return
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
		return
	}
	if host, has := ser.hosts[host_info[0]]; has {
		log.Println("rewrite host:", req.Host, "==>", host)
		req.URL.Host = host
		return
	}
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
				uValues.Add(k, fmt.Sprintf("%s", v))
			}
		default:
			log.Println("unkonw type:", value)
		}
	}
	return uValues
}

package serve

import (
	"net/http"
	"log"
	"fmt"
)
var rewriteJsTpl="function pproxy_rewrite(req){\n%s\nreturn req;\n}"

func (ser *ProxyServe)parseAndSaveRewriteJs(jsStr string) error{
	   rewriteJs:=fmt.Sprintf(rewriteJsTpl,jsStr)
	   js.Run(rewriteJs)
	   jsFn,err:=js.Get("pproxy_rewrite")
	   if(err==nil){
	      ser.RewriteJs=jsStr
	      ser.RewriteJsFn=jsFn
	   }
	   return err
}

func (ser *ProxyServe) reqRewrite(req *http.Request) {
   if(ser.RewriteJs!=""){
      urlObj, _ := js.Object(`ul={}`)
      urlObj.Set("url",req.URL.String())
      urlObj.Set("schema",req.URL.Scheme)
      urlObj.Set("host",req.URL.Host)
      urlObj.Set("path",req.URL.Path)
      urlObj.Set("rawquery",req.URL.RawQuery)
      urlObj.Set("fragment",req.URL.Fragment)
      urlObj.Set("opaque",req.URL.Opaque)
      username:=""
      psw:=""
      if(req.URL.User!=nil){
	      username=req.URL.User.Username()
	      psw,_=req.URL.User.Password()
      }
      urlObj.Set("username",username)
      urlObj.Set("password",psw)
      
      js_ret,err_js:=ser.RewriteJsFn.Call(ser.RewriteJsFn,urlObj)
      
      if(err_js==nil ){
	      if(js_ret.IsObject()){
          	obj,export_err:=js_ret.Export()
          	if(export_err==nil){
             	url_obj:=obj.(map[string]interface{})
             	url_new:=fmt.Sprintf("%s",url_obj["schema"])+"://";
             	username:=fmt.Sprintf("%s",url_obj["username"])
             	if(username!=""){
                	url_new+=fmt.Sprintf("%s:%s@",username,url_obj["password"])
             	}
             	url_new+=fmt.Sprintf("%s%s",url_obj["host"],url_obj["path"])
             	
             	if(url_new==req.URL.String()){
             	   return
             	    }
             	
			    var url_err error
		        req.URL,url_err=req.URL.Parse(url_new)
		        if(url_err!=nil){
		           log.Println("js filter err:",js_ret,url_err)
			       }
          	}else{
          	   log.Println("js filter result wrong",js_ret.String())
          	}
	        }
      }else{
          log.Println("js filter err:",err_js,js_ret)
        }
      
   }
}
package serve

import (
    "net/http"
    "strings"
    "strconv"
    "fmt"
    "net/url"
)

func (ser *ProxyServe)req_redo(w http.ResponseWriter, req *http.Request,values map[string]interface{}){
   if(req.Method=="POST"){
      ser.req_redoPost(w,req,values)
      return
   }
    docid_str:=strings.TrimSpace(req.FormValue("id"))
    if(docid_str==""){
       w.WriteHeader(http.StatusBadRequest)
       w.Write([]byte("empty id param"))
       return
    }
    docid, err_int := strconv.ParseUint(docid_str, 10, 64)
    if err_int != nil {
         w.WriteHeader(http.StatusInternalServerError)
         w.Write([]byte(fmt.Sprintf("param id[%s] error:\n%s",docid_str,err_int)))
         return
    }
    req_doc := ser.GetRequestByDocid(docid)
    if(req_doc==nil){
      w.WriteHeader(http.StatusNotFound)
      w.Write([]byte("request doc not found!"))
      return
    }
    fmt.Println(req_doc)
    _url:=fmt.Sprintf("%s",req_doc["url"])
    u,err:=url.Parse(_url)
    if(err!=nil){
         w.WriteHeader(http.StatusInternalServerError)
         w.Write([]byte(fmt.Sprintf("parse url[%s] error\n%s",_url,err)))
         return
    }
    u.RawQuery="";
    values["req"]=req_doc
    values["action_url"]=u.String()
    values["subTitle"]="redo|"+u.String()+"|"
    html:=render_html("redo.html",values,true)
    w.Write([]byte(html))
}

var redo_skip_headers=map[string]int{"Content-Length":1}


func (ser *ProxyServe)req_redoPost(w http.ResponseWriter, req *http.Request,values map[string]interface{}){
   redo:=req.FormValue("redo")
   basic:=make(map[string]string)
   basic["action_url"]=strings.TrimSpace(req.FormValue("basic_action_url"))
   method:=strings.TrimSpace(strings.ToUpper(req.FormValue("basic_method")))
   basic["method"]=method
   
//   basic_client_ip:=req.FormValue("basic_client_ip")
   
   header:=GetFormValuesWithPrefix(req.Form,"header_")
   get:=GetFormValuesWithPrefix(req.Form,"get_")
   post:=GetFormValuesWithPrefix(req.Form,"post_")

   formData:=make(map[string]interface{})
   formData["basic"]=basic
   
   formData["header"]=header
   formData["get"]=get
   formData["post"]=post
   
   values["form"]=formData   
   if(redo=="direct"){
     html:=render_html("redo_direct.html",values,true)
     w.Write([]byte(html))
     return
   }else{
	    req_bd:="";
	    
	    if(method=="POST"){
	       form_values:=make(url.Values)
	        for k,v:=range post{
		        for _,_v:=range v{
			      form_values.Add(k,_v)
			    }
		    }
		    req_bd=form_values.Encode()
	    }else{
	       form_values:=make(url.Values)
	        for k,v:=range get{
		        for _,_v:=range v{
			      form_values.Add(k,_v)
			    }
		    }
		    req_bd=form_values.Encode()
	    }
	    
	    redo_req,err:=http.NewRequest(method,basic["action_url"],strings.NewReader(req_bd))
	    if(err!=nil){
	       w.Write([]byte("build request failed\n"+err.Error()))
	      return
	    }
	    
//	    redo_req.RemoteAddr=basic_client_ip

	    for k,v:=range header{
	       if _,has:=redo_skip_headers[k];has{
	         continue
	       }
		    redo_req.Header.Set(k,strings.Join(v,";"))
	    }
	    ser.Goproxy.ServeHTTP(w, redo_req)
   }
}
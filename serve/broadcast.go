package serve

import (
    "fmt"
    "net/http"
    "path/filepath"
    "strings"
)

func (ser *ProxyServe) Broadcast_Req(req *http.Request, id int64, docid uint64, user string) bool {
    data := make(map[string]interface{})
    data["docid"] = fmt.Sprintf("%d", docid)
    data["sid"] = id % 1000
    data["host"] = req.Host
    data["client_ip"] = req.RemoteAddr
    data["path"] = req.URL.Path
    if(req.Method=="CONNECT"){
       data["path"]="https req,unknow path"
    }
    data["method"] = req.Method
    ser.mu.RLock()
    defer ser.mu.RUnlock()
    hasSend := false
    for _, client := range ser.wsClients {
        if(ser.conf.SessionView==SessionView_IP_FILTER && len(client.filter_ip)==0){
           continue
        }
        if (user=="guest") && checkFilter(req, client,user) {
            send_req(client, data)
            hasSend = true
        }
    }
    return hasSend
}


var extTypes map[string][]string = map[string][]string{
    "js":    []string{"js"},
    "css":   []string{"css"},
    "image": []string{"jpg", "jpeg", "png", "gif", "bmp", "tiff", "jpe", "tif", "webp", "ico"},
}

func checkFilter(req *http.Request, client *wsClient,user string) bool {
    if(len(client.filter_user)>0){
       user_in_list:=false
       for _,name:=range client.filter_user{
          if(name!="" && name==user){
             user_in_list=true
             break
          }
       }
       if(!user_in_list){
          return false
       }
    }
    
    if(len(client.filter_ip)>0){
      addr_info:=strings.Split(req.RemoteAddr,":")
      ip_in_list:=false;
      for _,ip:=range client.filter_ip{
         if(ip!="" && addr_info[0]==ip){
            ip_in_list=true
            break
         }
      }
      if(!ip_in_list){
         return false;
      }
    }
    
    if len(client.filter_url) > 0 {
        url := req.URL.String()
        has_kw := false
        for _, subUrl := range client.filter_url {
            if strings.Contains(url, subUrl) {
                has_kw = true
                break
            }
        }
        if !has_kw {
            return false
        }
    }
    
    if len(client.filter_hide_ext) > 0 {
        ext := strings.ToLower(strings.Trim(filepath.Ext(req.URL.Path), "."))
        for _, hide_type := range client.filter_hide_ext {
            for _, hide_ext := range extTypes[hide_type] {
                if ext == hide_ext {
                    return false
                }
            }
        }
    }
    if(len(client.filter_url_hide)>0){
        for _, hide_kw := range client.filter_url_hide {
           if hide_kw!="" && strings.Contains(req.URL.String(),hide_kw) {
                 return false
           }
        }
    }
    return true
}

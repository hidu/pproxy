package serve

import (
	"net/http"
	"path/filepath"
	"strings"
	"fmt"
)

func (ser *ProxyServe) Broadcast_Req( req *http.Request, id int64,docid uint64,user string) {
	data := make(map[string]interface{})
	data["docid"] = fmt.Sprintf("%d", docid)
	data["sid"] = id % 1000
	data["host"] = req.Host
	data["client_ip"] = req.RemoteAddr
	data["path"] = req.URL.Path
	data["method"] = req.Method
	ser.mu.RLock()
	defer ser.mu.RUnlock()
	fmt.Println("now broadcast")
	for _, client := range ser.wsClients {
		if(client.user==user && checkFilter(req,client)){
			send_req(client, data)
		}
	}
}
var extTypes map[string][]string=map[string][]string{
"js":[]string{"js"},
"css":[]string{"css"},
"image":[]string{"jpg","jpeg","png","gif","bmp","tiff","jpe","tif","webp","ico"},
}

func checkFilter(req *http.Request,client *wsClient) bool{
   if(client.filter_client_ip!="" && !strings.Contains(req.RemoteAddr,client.filter_client_ip)){
     return false
    }
   if(len(client.filter_url)>0){
	   url:=req.URL.String()
       has:=false
       for _,subUrl:=range client.filter_url{
           if(strings.Contains(url,subUrl)){
              has=true
              break
              }
         }
       if(!has){
           return false
         }
    }
   if(len(client.filter_hide)>0){
       ext:=strings.ToLower(strings.Trim(filepath.Ext(req.URL.Path),"."))
       for _,hide_type:=range client.filter_hide{
           for _,hide_ext:=range extTypes[hide_type]{
                if(ext==hide_ext){
                    return false
                      }
               }
         }
    }
	return true
}
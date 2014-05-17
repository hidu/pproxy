package serve
import (
  "net/http"
  "github.com/hidu/goutils"
	  "text/template"
)


func handleLocalReq(w http.ResponseWriter, req *http.Request){
   msg:=goutils.DefaultResource.Load("/res/tpl/index.html")
   tpl,_:=template.New("page").Parse(string(msg))
   values :=make(map[string]string)
   values["title"]=""
   values["version"]="0.1"
   tpl.Execute(w,values)
}
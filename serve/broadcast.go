package serve

import (
	"fmt"
	"net/http"
)

func (ser *ProxyServe) Broadcast_Req(req *http.Request, reqCtx *requestCtx) bool {
	data := make(map[string]interface{})
	data["docid"] = fmt.Sprintf("%d", reqCtx.Docid)
	data["sid"] = reqCtx.SessionId % 10000
	data["host"] = req.Host
	data["client_ip"] = req.RemoteAddr
	urlPath := req.URL.Path
	if req.URL.RawQuery != "" {
		urlPath += "?" + req.URL.RawQuery
	}
	data["path"] = urlPath
	data["url"] = req.URL.String()
	if req.Method == "CONNECT" {
		data["path"] = "https req,unknow path"
	}
	data["method"] = req.Method
	data["replay"] = reqCtx.IsRePlay

	hasSend := ser.wsSer.broadcastReq(req, reqCtx, data)
	return hasSend
}

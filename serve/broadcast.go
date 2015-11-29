package serve

import (
	"fmt"
)

// broadcastReq broadcast request to user's browser
func (ser *ProxyServe) broadcastReq(reqCtx *requestCtx) bool {
	req := reqCtx.Req
	data := make(map[string]interface{})
	data["docid"] = fmt.Sprintf("%d", reqCtx.Docid)
	data["sid"] = reqCtx.SessionID % 10000
	data["host"] = req.Host
	data["client_ip"] = req.RemoteAddr
	urlPath := req.URL.Path
	if req.URL.RawQuery != "" {
		urlPath += "?" + req.URL.RawQuery
	}
	data["path"] = urlPath
	data["url"] = req.URL.String()
	if req.Method == "CONNECT" && !ser.conf.SslOn {
		data["path"] = "https req,unknow path"
	}
	data["method"] = req.Method
	data["replay"] = reqCtx.IsRePlay

	hasSend := ser.wsSer.broadcastReq(req, reqCtx, data)
	return hasSend
}

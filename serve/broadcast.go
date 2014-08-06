package serve

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

func (ser *ProxyServe) Broadcast_Req(req *http.Request, reqCtx *requestCtx) bool {
	data := make(map[string]interface{})
	data["docid"] = fmt.Sprintf("%d", reqCtx.Docid)
	data["sid"] = reqCtx.SessionId % 1000
	data["host"] = req.Host
	data["client_ip"] = req.RemoteAddr
	data["path"] = req.URL.Path
	if req.Method == "CONNECT" {
		data["path"] = "https req,unknow path"
	}
	data["method"] = req.Method
	data["redo"] = reqCtx.IsReDo

	ser.mu.RLock()
	defer ser.mu.RUnlock()
	hasSend := false
	for _, client := range ser.wsClients {
		if ser.conf.SessionView == SessionView_IP_OR_USER && len(client.filter_ip) == 0 && len(client.filter_user) == 0 {
			continue
		}

		if reqCtx.User.Name != "" && len(client.filter_user) < 1 {
			continue
		}

		if checkFilter(req, client, reqCtx) {
			send_req(client, data)
			hasSend = true
		}
	}
	return hasSend
}

var extTypes map[string][]string = map[string][]string{
	"js":    {"js"},
	"css":   {"css"},
	"image": {"jpg", "jpeg", "png", "gif", "bmp", "tiff", "jpe", "tif", "webp", "ico", "webp"},
}

func checkFilter(req *http.Request, client *wsClient, reqCtx *requestCtx) bool {
	if len(client.filter_user) > 0 {
		user_in_list := false
		for _, name := range client.filter_user {
			if name == "any" && client.LoginUser != nil && client.LoginUser.IsAdmin {
				user_in_list = true
				break
			}
			if name != "" && name == reqCtx.User.Name {
				user_in_list = true
				break
			}
		}
		if !user_in_list {
			return false
		}
	}

	if len(client.filter_ip) > 0 {
		addr_info := strings.Split(reqCtx.RemoteAddr, ":")
		ip_in_list := false
		for _, ip := range client.filter_ip {
			if ip != "" && addr_info[0] == ip {
				ip_in_list = true
				break
			}
		}
		if !ip_in_list {
			return false
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
	if len(client.filter_url_hide) > 0 {
		_url := req.URL.String()
		for _, hide_kw := range client.filter_url_hide {
			if hide_kw != "" && strings.Contains(_url, hide_kw) {
				return false
			}
		}
	}
	return true
}

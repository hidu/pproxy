package serve

import (
	"gopkg.in/hidu/go-socket.io.v1"
	"net/http"
	"path/filepath"
	"strings"
)

type wsClient struct {
	ns            *socketio.NameSpace
	user          string
	filterUser    []string
	filterIP      []string
	filterHideExt []string
	filterURL     []string
	filterURLHide []string
	LoginUser     *User
}

var extTypes = map[string][]string{
	"js":    {"js"},
	"css":   {"css"},
	"image": {"jpg", "jpeg", "png", "gif", "bmp", "tiff", "jpe", "tif", "webp", "ico", "webp"},
}

func (client *wsClient) checkFilter(req *http.Request, reqCtx *requestCtx) bool {
	if len(client.filterUser) > 0 {
		userInList := false
		for _, name := range client.filterUser {
			if name == "any" && client.LoginUser != nil && client.LoginUser.IsAdmin {
				userInList = true
				break
			}
			if name != "" && name == reqCtx.User.Name {
				userInList = true
				break
			}
		}
		if !userInList {
			return false
		}
	}

	if len(client.filterIP) > 0 {
		addrInfo := strings.Split(reqCtx.RemoteAddr, ":")
		ipInList := false
		for _, ip := range client.filterIP {
			if ip != "" && addrInfo[0] == ip {
				ipInList = true
				break
			}
		}
		if !ipInList {
			return false
		}
	}

	if len(client.filterURL) > 0 {
		url := req.URL.String()
		hasKw := false
		for _, subURL := range client.filterURL {
			if strings.Contains(url, subURL) {
				hasKw = true
				break
			}
		}
		if !hasKw {
			return false
		}
	}

	if len(client.filterHideExt) > 0 {
		ext := strings.ToLower(strings.Trim(filepath.Ext(req.URL.Path), "."))
		for _, hideType := range client.filterHideExt {
			for _, hideExt := range extTypes[hideType] {
				if ext == hideExt {
					return false
				}
			}
		}
	}
	if len(client.filterURLHide) > 0 {
		_url := req.URL.String()
		for _, hideKw := range client.filterURLHide {
			if hideKw != "" && strings.Contains(_url, hideKw) {
				return false
			}
		}
	}
	return true
}

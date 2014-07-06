package serve

import (
	"encoding/base64"
	"github.com/hidu/goutils"
	"net/http"
	"strings"
)

var proxyAuthorizatonHeader = "Proxy-Authorization"

func getAuthorInfo(req *http.Request) *User {
	defaultInfo := new(User)
	authheader := strings.SplitN(req.Header.Get(proxyAuthorizatonHeader), " ", 2)
	if len(authheader) != 2 || authheader[0] != "Basic" {
		return defaultInfo
	}
	userpassraw, err := base64.StdEncoding.DecodeString(authheader[1])
	if err != nil {
		return defaultInfo
	}
	userpass := strings.SplitN(string(userpassraw), ":", 2)
	if len(userpass) != 2 {
		return defaultInfo
	}
	return &User{Name: userpass[0], Psw: utils.StrMd5(userpass[1])}
}

func (ser *ProxyServe) CheckUserLogin(userInfo *User) bool {
	if userInfo == nil || ser.Users == nil {
		return false
	}

	if userInfo.SkipCheckPsw {
		return true
	}

	if user, has := ser.Users[userInfo.Name]; has {
		return user.Psw == userInfo.Psw
	}
	return false
}

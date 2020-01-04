package serve

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/hidu/goutils/str_util"
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
	return &User{Name: userpass[0], PswMd5: str_util.StrMd5(userpass[1]), Psw: userpass[1]}
}

func (ser *ProxyServe) checkUserLogin(userInfo *User) bool {
	if userInfo == nil || ser.Users == nil {
		return false
	}

	if userInfo.SkipCheckPsw {
		return true
	}

	if user, has := ser.Users[userInfo.Name]; has {
		return user.PswMd5 == userInfo.PswMd5
	}
	return false
}

// (ser.conf.AuthType == AuthType_Basic && !ser.CheckUserLogin(reqCtx.User))
func (ser *ProxyServe) checkHTTPAuth(reqCtx *requestCtx) bool {
	switch ser.conf.AuthType {
	case authTypeNO:
		return true
	case authTypeBasic:
		return ser.checkUserLogin(reqCtx.User)
	case authTypeBasicWithAny:
		return reqCtx.User.Name != ""
	case authTypeBasicTry:
		if reqCtx.ClientSession.RequestNum == 1 {
			return reqCtx.User.Name != ""
		}
		return true
	default:
		return false
	}
	return false
}

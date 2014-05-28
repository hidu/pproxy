package serve

import (
	"encoding/base64"
	"net/http"
	"strings"
)

type BasicUserInfo struct {
	Name string
	Psw  string
}

func (info *BasicUserInfo) isEqual(name, psw string) bool {
	return info.Name == name && info.Psw == psw
}

var proxyAuthorizatonHeader = "Proxy-Authorization"

func getAuthorInfo(req *http.Request) *BasicUserInfo {
	authheader := strings.SplitN(req.Header.Get(proxyAuthorizatonHeader), " ", 2)
	if len(authheader) != 2 || authheader[0] != "Basic" {
		return nil
	}
	userpassraw, err := base64.StdEncoding.DecodeString(authheader[1])
	if err != nil {
		return nil
	}
	userpass := strings.SplitN(string(userpassraw), ":", 2)
	if len(userpass) != 2 {
		return nil
	}
	return &BasicUserInfo{Name: userpass[0], Psw: userpass[1]}
}

func (ser *ProxyServe)CheckUserLogin(userInfo *BasicUserInfo) bool{
    if (userInfo==nil){
      return false
    }
    for name,psw:=range ser.Users{
       if(userInfo.isEqual(name,psw)){
          return true
       }
    }
    return false;
}
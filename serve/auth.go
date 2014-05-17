package serve

import (
  "net/http"
  "encoding/base64"
  "strings"
)


type BasicUserInfo struct{
  Name string
  Psw string
}


func (info *BasicUserInfo)isEqual(name,psw string) bool{
   return info.Name==name && info.Psw==psw
}
var proxyAuthorizatonHeader = "Proxy-Authorization"

func getAuthorInfo(req *http.Request) *BasicUserInfo{
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
	return &BasicUserInfo{Name:userpass[0],Psw:userpass[1]}
}
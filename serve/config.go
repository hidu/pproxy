package serve

import (
	"encoding/json"
	"github.com/hidu/goutils"
	"log"
	"strings"
)

type Config struct {
	Port         int    `json:"port"`
	Title        string `json:"title"`
	Notice       string `json:"notice"`
	AuthType     int    `json:"authType"`
	DataDir      string `json:"dataDir"`
	ResponseSave int    `json:"responseSave"`
	SessionView  int    `json:"sessionView"`
}

const (
	AuthType_NO            = 0
	AuthType_Basic         = 1
	AuthType_Basic_WithAny = 2
	AuthType_Basic_Try     = 3

	ResponseSave_All      = 0
	ResponseSave_HasBroad = 1 //has show

	SessionView_ALL        = 0
	SessionView_IP_OR_USER = 1
)

type User struct {
	Name         string
	Psw          string //md5 encode
	IsAdmin      bool
	SkipCheckPsw bool
}

const (
	Content_Encoding = "Content-Encoding"
)

//"0:no auth | 1:basic auth | 2:basic auth with any name"

func getVersion() string {
	return string(utils.DefaultResource.Load("/res/version"))
}

func (u *User) isPswEq(psw string) bool {
	return u.Psw == utils.StrMd5(psw)
}

func LoadConfig(confPath string) (*Config, error) {
	data, err := utils.File_get_contents(confPath)
	if err != nil {
		log.Println("load config", confPath, "failed,err:", err)
		return nil, err
	}
	var config *Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Println("config is  incorrect", err)
		return nil, err
	}
	return config, nil
}

type configHosts map[string]string

func loadHosts(confPath string) (hosts configHosts, err error) {
	hosts = make(configHosts)
	if !utils.File_exists(confPath) {
		return
	}
	hosts_byte, err := utils.File_get_contents(confPath)
	if err != nil {
		log.Println("load hosts_file failed:", confPath, err)
		return nil, err
	}
	hosts_arr := utils.LoadText2Slice(string(hosts_byte))
	for _, v := range hosts_arr {
		if len(v) != 2 {
			log.Println("hosts file line wrong,ignore,", v)
			continue
		}
		hosts[v[0]] = v[1]
	}
	return
}

func loadUsers(confPath string) (users map[string]*User, err error) {
	users = make(map[string]*User)
	if !utils.File_exists(confPath) {
		return
	}
	userInfo_byte, err := utils.File_get_contents(confPath)
	if err != nil {
		log.Println("load user file failed:", confPath, err)
		return
	}
	lines := utils.LoadText2Slice(string(userInfo_byte))
	for _, line := range lines {
		if len(line) < 2 {
			log.Println("skip user file,line:", line)
			continue
		}
		isAdmin := len(line) > 2 && line[2] == "admin"
		psw := line[1]
		if strings.HasSuffix(psw, ":md5") {
			if len(psw) == 36 {
				psw = psw[:32]
			} else {
				log.Println("user config wrong", line)
			}
		} else {
			psw = utils.StrMd5(psw)
		}
		users[line[0]] = &User{Name: line[0], Psw: psw, IsAdmin: isAdmin}
	}
	return
}

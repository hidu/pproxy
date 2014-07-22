package serve

import (
	"fmt"
	"github.com/hidu/goutils"
	"log"
	"strings"
	"github.com/Unknwon/goconfig"
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

func (u *User) String() string {
	return fmt.Sprintf("Name:%s,Psw:%s,isAdmin:%v,SkipCheckPsw:%v", u.Name, u.Psw, u.IsAdmin, u.SkipCheckPsw)
}

const (
	Content_Encoding = "Content-Encoding"
)

//"0:no auth | 1:basic auth | 2:basic auth with any name"

func GetVersion() string {
	return string(utils.DefaultResource.Load("/res/version"))
}

func GetDemoConf()string{
	return strings.TrimSpace(string(utils.DefaultResource.Load("/res/conf/demo.conf")))
}


func (u *User) isPswEq(psw string) bool {
	return u.Psw == utils.StrMd5(psw)
}

func LoadConfig(confPath string) (*Config, error) {
    gconf,err:=goconfig.LoadConfigFile(confPath)
	if err != nil {
		log.Println("load config", confPath, "failed,err:", err)
		return nil, err
	}
	config:=new(Config)
	config.Port=gconf.MustInt(goconfig.DEFAULT_SECTION,"port",8080)
	config.Title=gconf.MustValue(goconfig.DEFAULT_SECTION,"title")
	config.Notice=gconf.MustValue(goconfig.DEFAULT_SECTION,"notice")
	config.DataDir=gconf.MustValue(goconfig.DEFAULT_SECTION,"dataDir")
	
	_authType:=strings.ToLower(gconf.MustValue(goconfig.DEFAULT_SECTION,"authType","none"))
	authTypes:=map[string]int{"none":0,"basic":1,"try_basic":2}
	
	hasError:=false
	if authType,has:=authTypes[_authType];has{
	  config.AuthType=authType
	}else{
	   hasError=true
	   log.Println("conf error,unknow value authType:",_authType)
	}
	
	_responseSave:=strings.ToLower(gconf.MustValue(goconfig.DEFAULT_SECTION,"responseSave","all"))
	responseSaveMap:=map[string]int{"all":0,"only_broadcast":1}
	
	if responseSave,has:=responseSaveMap[_responseSave];has{
	  config.ResponseSave=responseSave
	}else{
		 hasError=true
	     log.Println("conf error,unknow value responseSave:",_authType)
	}
	
	_sessionView:=strings.ToLower(gconf.MustValue(goconfig.DEFAULT_SECTION,"sessionView","all"))
	sessionViewMap:=map[string]int{"all":0,"ip_or_user":1}
	
	if sessionView,has:=sessionViewMap[_sessionView];has{
	  config.SessionView=sessionView
	}else{
	     hasError=true
	     log.Println("conf error,unknow value responseSave:",_authType)
	}
	
	if(hasError){
	   return config,fmt.Errorf("config error")
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

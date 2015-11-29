pproxy 0.5
======
##intro
http抓包代理程序,http协议调试工具。  
采用golang编写，采用bs模式(s-代理程序，b-会话查看、配置管理等功能)  

0.4.2版本已经支持websocket代理，以及重定向(和普通http请求一样使用)  

0.5 版本是对底层存储进行了替换，并且尝试支持https抓包

##install
下载编译好的可执行文件: <http://pan.baidu.com/s/1i3pAe7V>  

已经安装golang的用户直接安装：  
>go get -u github.com/hidu/pproxy

##功能特性
<pre>
1.url重定向
   如把 http://www.baidu.com/s?wd=pproxy 修改为 http://m.baidu.com/s?wd=pproxy
   或者把 ws://www.test.com/a 重定向到 ws://www.example.com/b
   
2.form表单动态修改  
   get、post可以动态修改（增删改）  
   
3.hosts文件支持
  相当于 修改host或者dns 如  
  将www.baidu.com 请求全部发往127.0.0.1  
  将www.baidu.com:81 请求全部发往192.168.1.2:8080  
  
4.可查看request 和response详情
   form表单参数，header等都可以很方便的看到
   
5.登录认证支持
   支持httpBasic认证
   
6.replay功能
   可以修改request的参数（get、post、header）

7.父级代理
  
</pre>

##配置

###rewrite req
使用javascript来配置重定向功能，如
```
if(req.host=="www.baidu.com"){
   req.host="www.163.com"
   req.host_addr="127.0.0.0:81" // send req to 127.0.0.1:81
}
```
当然也可以这样：
```
if(req.host.indexOf("baidu.com")>-1){
  req.host_addr="127.0.0.0:81"
}
```

###req变量示例
```
#url : http://www.example.com/album/list?cid=126
#req对象有如下一下属性：
schema : http
host : www.example.com
port : 80
path : /album/list
get: {cid:[123]}
post: {}
username : 
password : 
method: GET
form_get  : {add:function(k,v){},set:function(k,v){},get:function(k){},len:function(){}} 
form_post : {add:function(k,v){},set:function(k,v){},get:function(k){},len:function(){}}

host_addr: #修改该请求的host是使用，如 127.0.0.1:3218

#注意 get 和post的值是数组，如上cid参数
#form_get 用于更方便的操作  get参数对象
#form_post 用于更方便的操作 post参数对象
```

###hosts
增强的hosts文件使用:
```
www.baidu.com 127.0.0.1
www.baidu.com:81 10.0.2.2:8080
```

###other
忽略禁用req_rewrite.js  
在js文件的第一行内容写入 ```//ignore```

req_rewrite.js支持不同用户设置不同的规则。默认使用当前验证使用用户名的规则，若无则使用默认的。  

###配置文件结构：
```
conf/
├── pproxy.conf          #server的配置
├── hosts_8080           #8080端口server的hosts规则
├── req_rewrite_8080.js  #8080端口server的url重写规则
├── hosts_8081
├── req_rewrite_8081.js
└── users                #全局帐号配置文件
```

###users配置:
```
#帐号 admin，密码 是 psw,是管理员帐号
name:admin psw:psw is_admin:admin

#密码也可以存储为md5值，使用  psw_md5：32位md密码
name:admin_sec psw_md5:7bb483729b5a8e26f73e1831cde5b842 is_admin:admin
```
可以在线修改配置时必须使用管理员帐号登录

###配置文件示例(pproxy.conf):
```
#提供代理服务的端口
port : 8080

#管理界面的端口，为0表示和代理服务使用相同的端口,eg:8081
adminPort : 0

title : demo
notice :notice notice

#数据存放目录，相对于当前配置的路径
dataDir : ../data/

#数据存放天数，0为永久存储（目前只在重启的时候会进行数据清理）
dataStoreDay : 15

#代理服务认证方式
#options:{none : 无认证,basic:http basic ,basic_try:尝试httpBasic认证 ,basic_any:任意帐号}
authType : none

#那些request和response数据进行存储
#options:{ all : 所有   only_broadcast : 发送到session list的才存储}
responseSave : only_broadcast

#session列表查看数据
# options :{ all:所有人可见 ip_or_user : 输入正确的ip或者user后可见}
sessionView : all

#父级代理
#eg http://10.10.2.2:3128 or http://name:psw@10.10.2.2:3128
# http://pass:pass@10.10.2.2:3128 the user and psw will pass through to the parent proxy
parentProxy:


#是否使用中间人方式对https进行抓包，若启用的话 需要客户端按照证书-/res/private/client_cert.pem
#pproxy内置默认证书存放在/res/private目录中
#options:{on:启用  off:禁用}
ssl : on

#ssl 服务端秘钥文件地址，为空则使用默认内置的 /res/private/server_key.pem
ssl_server_key: 
#ssl 公钥地址 ，为空则使用默认内置的 /res/private/client_cert.pem
ssl_client_cert :
```

##(管理)web查看界面
方式1： 直接访问 http://serverHost:port  
方式2： 直接访问 http://serverHost:adminPort  
方式3： 浏览器设置http代理 serverHost:port，访问 http://pproxy.man 或者 http://pproxy.com  



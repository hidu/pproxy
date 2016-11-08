pproxy 0.5.2
======
HTTP protocol analysis tool.  
write by golang,with BS architecture. 


download binary file (linux & windows): <http://pan.baidu.com/s/1i3pAe7V>  

features：
<pre>
1.url redirect
   redirect *http://www.baidu.com/s?wd=pproxy* to  *http://m.baidu.com/s?wd=pproxy*
   redirect  *ws://www.test.com/a* to  *ws://www.example.com/b*
   
2.form dynamic modification  
   get、post and header all can modify  
   
3.hosts
  www.baidu.com to 127.0.0.1  
  or www.baidu.com:81 to 192.168.1.2:8080 ,and only  takes effect on port 81  
  
4.view request and response detail
   form params，header and all response and easy to share
   
5.auth sup
   http Basic or only try basic auth at first request
   
6.replay
   can modify the get、post、header params and replay the request

7.parent proxy
  
</pre>

use javascript code as config to modify the request params:
```
if(req.host=="www.baidu.com"){
   req.host="www.163.com"
   req.host_addr="127.0.0.0:81" // send req to 127.0.0.1:81
}
```
or：
```
if(req.host.indexOf("baidu.com")>-1){
  req.host_addr="127.0.0.0:81"
}
```

request params dump：
```
#url : http://www.example.com/album/list?cid=126
#request has these attrs：
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

host_addr: #modify hosts eg:127.0.0.1:3218

#note get and post value is array
#form_get: helper function for get params
#form_post: helper function for post params
```


hosts config demo:
```
www.baidu.com 127.0.0.1
www.baidu.com:81 10.0.2.2:8080
```

disable req_rewrite.js  
first line ```//ignore```


req_rewrite.js支持不同用户设置不同的规则。默认使用当前验证使用用户名的规则，若无则使用默认的。  

configs：
```
conf/
├── pproxy.conf          #server config
├── hosts_8080           #hosts for 8080
├── req_rewrite_8080.js  #8080端口server的url重写规则
├── hosts_8081
├── req_rewrite_8081.js
└── users                #全局帐号配置文件
```

users配置:
```
#帐号 admin，密码 是 psw,是管理员帐号
name:admin psw:psw is_admin:admin

#密码也可以存储为md5值，使用  psw_md5：32位md密码
name:admin_sec psw_md5:7bb483729b5a8e26f73e1831cde5b842 is_admin:admin
```
可以在线修改配置时必须使用管理员帐号登录

配置文件示例:
```

port : 8080

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
```

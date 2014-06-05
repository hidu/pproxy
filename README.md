pproxy
======
为马农特别准备的代理http代理服务器

<pre>
支持：
1.url重定向
   如把 http://www.baidu.com/s?wd=pproxy 修改为 http://m.baidu.com/s?wd=pproxy
   
2.host ip重定向
  相当于 修改host或者dns 如将www.baidu.com 请求全部发往127.0.0.1
  
3.可查看request 和response详情
   form表单参数，header等都可以很方便的看到
   
4.登录认证支持
   支持httpBasic认证
</pre>
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

req变量示例：
```
url : http://wenku.baidu.com/album/list?cid=126
schema : http
host : wenku.baidu.com
port : 
path : /album/list
rawquery : cid=126
username : 
password : 
```
除了url变量外，其他的都是可以修改来对request进行重写的

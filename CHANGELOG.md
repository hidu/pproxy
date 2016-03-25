2015-03-25
    1.版本号升级为0.5.1
    2.改进request动态修改引擎并发异常问题
    
2014-12-27
	1.版本号升级为0.4.7
	2.添加adminPort配置项，以独立的端口提供给管理界面

2014-12-17
	1.静态资源使用goassest方式而不使用之前的读取zip的方式
	
2014-11-08
	1.重构代理处理逻辑，改进Upgrade代理协议
	2.会话详情页面展现请求时间
	3.upgrade结束的时候也记录一条response 以方便查看何时断开连接
	
2014-09-27
    1.http session list support local filter
    
    
2014-09-14
    1.downgrade the socket.io lib
    
2014-08-14
   1.emit data with base64encode
   2.fix some url has no schema
   
2014-08-10
   1.websocket proxy support
   
2014-08-06
   1.update socket.io
   
2014-07-19
   1.修复监听端口为80时不能查看会话列表的问题
   2.完善帮助说明

2014-07-15
   1.get和post参数支持重写
   2.重写请求出现错误自己返回502错误

2014-07-12 
   1.认证机制升级，新认证机制：一个ip第一次访问的时候会要求登录，若没有输入登录信息也跳过。  
   2.管理员用户（登录后）在session filter 输入user：any 可以查看到所有的会话信息  

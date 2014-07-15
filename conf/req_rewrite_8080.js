if(req.host=="news.163.com"){
	req.host="news.baidu.com"
        req.host_addr="127.0.0.1:80"
}
console.log(form_get)
form_get.add("b[]","world")
form_get.add("b[]","你好")
form_post.add("c[]","你好")
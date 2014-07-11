function pproxy_rewrite(req){
	%s
//	console.log("get:",typeof req.form_get)
//	console.log("get_a:",req.form_get["a"])
//	console.log("post:",req.form_post)
	return req;
}
//
//var FormValue={};
//FormValue.prototype.Get=function(name){
//	var vs=this[name]||[]
//	if(vs.length>0){
//		return vs[vs.length-1]
//	}
//}
//FormValue.prototype.Set=function(name,val){
//	this[name]=[val+""]
//}
//FormValue.prototype.Add=function(name,val){
//	
//}
//FormValue.prototype.Del=function(name,val){
//	
//}

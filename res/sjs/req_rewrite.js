
function pproxy_rewrite(req){
	req.get=pproxy_params_copy(req._pproxy_get)
	req.post=pproxy_params_copy(req._pproxy_post)
	
	req.PproxyChangeFlags={"get":0,"post":0}
	
	var form_get=pproxy_obj_helper(req.get,req.PproxyChangeFlags,"get");
	var form_post=pproxy_obj_helper(req.post,req.PproxyChangeFlags,"post");
	
	CUSTOM_JS
	return req;
}
//clone the get and post params
function pproxy_params_copy(obj){
    var newObj=new Object();
    for(var k in obj){
		var arr=new Array();
		for(var i in obj[k]){
			arr[i]=obj[k][i]+"";
		}
		newObj[k]=arr;
    }
    return newObj
} 

function pproxy_obj_helper(values,flagObj,type){
  return {
      get:function(name){
    	  return values[name]
      },
      set:function(name,val){
    	  values[name]=[val]
    	  flagObj[type]=1
      },
      add:function(name,val){
    	  val=val+""
    	  if(values[name]!=undefined){
    		  values[name].push(val)
    	  }else{
    		  values[name]=[val]
    	  }
    	  flagObj[type]=1
      },
      del:function(name){
    	  delete values[name]
    	  flagObj[type]=1
      },
      len:function(name){
    	  if(name==undefined){
    	    return values.length
    	  }else{
    		return (values[name]||[]).length 
    	  }
      }
  }	
}

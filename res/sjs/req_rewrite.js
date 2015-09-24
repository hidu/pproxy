
function pproxy_rewrite(req){
	req.get=pproxy_params_copy(req.origin.get||{});
	req.post=pproxy_params_copy(req.origin.post||{});
	
	
	for(var k in req.origin.header){
		req[k]=req.origin.header[k]+"";
	}
	
	var form_get=pproxy_obj_helper(req.get);
	var form_post=pproxy_obj_helper(req.post);
	
	var use_file=function(filePath){
		filePath=filePath+""
		req.url=filePath.substr(0,7)=="http://"?filePath:("http://PPROXY_HOST"+"/f/"+filePath)
	}
	
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

function pproxy_obj_helper(values){
  return {
      get:function(name){
    	  return values[name]
      },
      set:function(name,val){
    	  values[name]=[val]
      },
      add:function(name,val){
    	  val=val+""
    	  if(typeof values[name]=="undefined"){
    		  values[name]=[val]
    		  return
    	  }
    	  if(typeof values[name]!="Object"){
    		  values[name]=[values[name]+""]
    	  }
          values[name].push(val)
      },
      del:function(name){
    	  delete values[name]
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

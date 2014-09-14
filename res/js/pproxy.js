var socket = io.connect();
var connectNum=0;
var pproxy_colors=["#FFFFFF","#CCFFFF","#FFCCCC","#99CCCC","996699","#CC9999","#0099CC","#FFFF66","#336633","#99CC00"]

var ip_colors={} 
var pproxy_session_max_len=0;

var pproxy_req_list=[];

if(window.localStorage){
	pproxy_req_list=$.parseJSON(window.localStorage["reqs"]||"[]")
	pproxy_session_max_len=1000;
}

function pproxy_show_reqs_from_local(){
	for(var i=0;i<pproxy_req_list.length;i++){
		pproxy_show_req(pproxy_req_list[i])
	}
}

window.onbeforeunload=function(){
	if(pproxy_session_max_len>0){
		window.localStorage["reqs"]=JSON.stringify(pproxy_req_list)
	}
}

function pproxy_save_req_local(dataStr64){
	if(pproxy_session_max_len<1){
		return;
	}
	var n=pproxy_req_list.push(dataStr64)
	if(n>pproxy_session_max_len){
		pproxy_req_list.shift();
	}
}

function pporxy_session_clean(){
	pproxy_req_list=[]
	$('#tb_network tbody').empty();
}

function pproxy_log(msg){
	$("#log_div").append("<div>"+(new Date().toString())+":"+msg+"</div>");
}


socket.on('connect', function() {
	connectNum++
	if(connectNum>1){
		pproxy_log("ws error.connectNum="+connectNum);
		socket.emit("disconnect");
		return;
	}
    $("#connect_status").html("online")
    $("#network_filter_form").change();
});

socket.on("disconnect", function() {
	connectNum--;
    $("#connect_status").html("<font color=pink>offline</font>");
});

socket.on("hello",function(data){
	console && console.log(new Date().toString(),data)
});

function pproxy_getColor(addr){
	var info=addr.split(":")
	if(info.length!=2){
		return "#FFFFFF"
	}
	var ip=info[0];
	var color=ip_colors[ip]
	if(!color){
		var l=0;
		for(var _t in ip_colors){
			l++
		}
		color=pproxy_colors[l% pproxy_colors.length]
		ip_colors[ip]=color
	}
	return color
}


function pproxy_show_req(dataStr64) {
    var dataStr=Base64.decode(dataStr64);
	var data=$.parseJSON(dataStr)
	
    console && console.log("req", data)
    var html="<tr onclick=\"get_response(this,'" + data['docid'] + "')\" ";
    if(data["redo"]){
    	html+="class='redo' ";
    }
    html+=">" 
    + "<td bgcolor='"+pproxy_getColor(data["client_ip"])+"'>" + data["sid"] + "</td>"
    + "<td><div class='oneline' title='"+h(data["host"])+"'>" + data["host"] + "</div></td>" +
    "<td><div class='oneline' title='"+h(data["url"])+"'>" +data["method"]+"&nbsp;"+ h(data["path"])+ "</div></td>" + 
    "</tr>";
    $("#tb_network tbody").prepend(html);
}


socket.on("req",function(dataStr64){
	pproxy_save_req_local(dataStr64)
	pproxy_show_req(dataStr64)
});


socket.on("user_num", function(data) {
	$("#user_num_online").html(data);
});

socket.on("res",
        function(dataStr64) {
	        var dataStr=Base64.decode(dataStr64);
			var data=$.parseJSON(dataStr)
			console && console.log(data)
            var req = data["req"];
            var res = data["res"];
            var html="";
            if(req){
	            var re_do_str=req["schema"]=="http"?("&nbsp;<a target='_blank' href='/redo?id="+req["id"]+"'>redo</a>"):"";
	            
	            html += "<div><table class='tb_1'><caption>Request"+re_do_str+"</caption>";
	            html += "<tr><th width='80px'>url</th><td>" + h(req["url"]) + "</td></tr>"
	            if (req["url_origin"]!=req["url"]) {
	                html += "<tr><th>origin</th><td><span style='color:blue'>" + h(req["url_origin"]) + "</span></td></tr>";
	            }
	            if (req["msg"]) {
	            	html += "<tr><th>msg</th><td><span style='color:red'>" + h(req["msg"])+"</span></td></tr>";
	            }
	            html += "<tr><th>proxy_urer</th>" +
	            		"<td><b>remote_addr : </b>&nbsp;" +req["client_ip"] + "&nbsp;&nbsp;<b> docid : </b>&nbsp;"+ req["id"] + 
	            		"</td></tr>";
	            html += pproxy_tr_sub_table(req["form_get"], "get_params");
	            html += pproxy_tr_sub_table(req["form_post"], "post_params");
	            if (req["dump"]) {
	                html += "<tr><th>req_dump</th><td>" + h(Base64.decode(req["dump"])).replace(/\n/g, "<br/>")
	                        + "</td></tr>";
	            }
	            html += "</table></div>";
            }else{
            	html="<br/><br/><br/><br/><center>request not exists!</center>";
            }
            var res_link = "";
            
            if (res) {
                res_link = "<a href='/response?id=" + res["id"] + "' target='_blank'>view</a>";
                html += "<div><table class='tb_1'><caption>Response&nbsp;" + res_link + "</caption>"
            }
            
            var hideBigBody=false;
            
            if (res) {
                if (res["dump"]) {
                    html += "<tr><th width='80px'>res_dump</th><td>" + h(Base64.decode(res["dump"])).replace(/\n/g, "<br/>")
                    + "</td></tr>";
                }
                var body_str = Base64.decode(res["body"])
                var isImg = res["header"]["Content-Type"] != undefined
                        && res["header"]["Content-Type"][0].indexOf("image") > -1;

                var isStatusOk = res["status"] == 200;

                var bd_json = pproxy_parseAsjson(body_str);

                if (bd_json) {
                	hideBigBody=true;
                    html += "<tr><th width='80px'>body_json</th><td>" + bd_json + "</td></tr>";
                }
                if (isImg) {
                	hideBigBody=true;
                    html += "<tr><th>body_img</th><td><img src='data:" + res["header"]["Content-Type"][0] + ";base64,"
                            + res["body"] + "'/></td></tr>";
                }
                if (!isImg || res["body"].length < 1000 || !isStatusOk) {
                	html += "<tr><th width='80px'>body";
                	if(res["body"].length>400){
                		html+="<div><a href='#' onclick='return pproxy_res_td_body_toggle()'>toggle</a></div>";
                	}else{
                		hideBigBody=false;
                	}
                	html+= "</th>" +
                			"<td>" +
                			"<div id='res_td_body' "+(hideBigBody?"class='res_td_body' ":"")+">" + h(body_str).replace(/\n/g, "<br/>") + 
                			"</div></td></tr>";
                }
            }

            html += "</table></div>";
            $("#right_content").empty().html(html).hide().slideDown("fast");
        })

function pproxy_res_td_body_toggle(){
	$("#res_td_body").toggleClass("res_td_body");
	return false;
}
        
function pproxy_parseAsjson(str) {
    try {
    	str=str+""
    	if(str[0]!="{" && str[0]!="["){
    		return false;
    	}
        var jsonObj = JSON.parse(str);
        if (jsonObj) {
            var json_str = JSON.stringify(jsonObj, null, 4);
            return "<pre>" + json_str + "</pre>";
        }
    } catch (e) {
    }
    return false;
}

function pproxy_tr_sub_table(obj, name) {
    if (!obj) {
        return "";
    }
    var html = "<tr><th>" + name + "</th><td class='td_has_sub'><table class='tb_1'>";
    var i = 0;
    var max_key_len=0;
    for ( var k in obj) {
    	max_key_len=Math.max(max_key_len,(k+"").length);
    }
    for ( var k in obj) {
        html += "<tr><th  "+(max_key_len<40?"width='120px' nowrap":"width='140px'")+">" + k + "</th><td><ul class='td_ul'>";
        for ( var i in obj[k]) {
            html += "<li>";
            var json_str = pproxy_parseAsjson(obj[k][i]);
            if (json_str) {
                html += json_str;
            } else {
                html += h(obj[k][i]);
            }
            html += "</li>";
        }
        html += "</ul></td></tr>";
        i++
    }
    if (i < 1) {
        return "";
    }
    html += "</table></td></tr>"
    return html
}

function pproxy_show_response(docid){
	console && console.log("get_response docid=", docid);
    var loading_msg="loading...docid=" + docid;
    var isValidId=(docid+"").length>2;
    if(!isValidId){
    	loading_msg="https request:no data"
    }else{
    	loading_msg+="&nbsp;<a href='javascript:;' onclick=\"pproxy_show_response('"+docid+"')\">reload</a>";
    }
    $("#right_content").empty().html("<center style='margin:200px 0 auto'>"+loading_msg+"</center>");
    if(!isValidId){
    	return;
    }
	socket.emit("get_response", docid);
}

function get_response(tr, docid) {
    pproxy_show_response(docid);
    $(tr).parent("tbody").find("tr").removeClass("selected");
    $(tr).addClass("selected");
    location.hash="req_"+docid;
}

function bytesToString(bytes) {
    var result = "";
    for (var i = 0; i < bytes.length; i++) {
        result += String.fromCharCode(parseInt(bytes[i], 2));
    }
    return result;
}

function h(html) {
	if(html==""){
		return "&nbsp;";
	}
	html = (html+"").replace(/&/g, '&amp;')
				.replace(/</g, '&lt;')
				.replace(/>/g, '&gt;')
			    .replace(/'/g, '&acute;')
			    .replace(/"/g, '&quot;')
	            .replace(/\|/g, '&brvbar;');
    return html;
}

$().ready(function() {
	$("#network_filter_form input:text").each(function(){
		pproxy_local_save(this,$(this).attr("name"));
	});
	var filter_form=$("#network_filter_form");
	filter_form.change(function() {
        var form_data = $(this).serialize();
        socket.emit("client_filter", form_data);
    });
    
    setTimeout(function(){filter_form.change();},600);
    setTimeout(function(){filter_form.change();},3000);
    
	filter_form.find("input:text").keyup(function(){
		filter_form.change();
    });
    
    if(location.hash.match(/req_\d+/)){
        var docid=location.hash.substr(5);
        setTimeout((function(id){
        	return function(){
        		pproxy_show_response(id);
        	}
        })(docid),500);
    }
    
    setTimeout(pproxy_show_reqs_from_local,0);
});

function pproxy_local_save(target,id){
	if(!window.localStorage){
		return;
	}
	$(target).val(window.localStorage[id]||"").change(function(){
		window.localStorage[id]=$(this).val();
	});
}
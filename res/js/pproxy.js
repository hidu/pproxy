var socket = io.connect();
var connectNum=0;
socket.on('connect', function() {
	connectNum++
	if(connectNum>1){
		alert("ws error.connectNum="+connectNum);
		socket.emit("disconnect")
	}
    $("#connect_status").html("online")
    $("#network_filter_form").change();
});

socket.on("disconnect", function() {
	connectNum--
    $("#connect_status").html("offline")
});

socket.on("req", function(data) {
    console && console.log("req", data)
    var html="<tr onclick=\"get_response(this,'" + data['docid'] + "')\" ";
    if(data["redo"]){
    	html+="class='redo' "
    }
    html+=">" 
    + "<td>" + data["sid"] + "</td>"
    + "<td>" + data["host"] + "</td>" +
    "<td>" +data["method"]+"&nbsp;"+ data["path"] + "</td>" + 
    "</tr>"
    $("#tb_network tbody").prepend(html)
})
socket.on("res",
        function(data) {
            console && console.log("res", data)
            var req = data["req"];
            var res = data["res"];
            var re_do_str="&nbsp;<a target='_blank' href='/redo?id="+req["@id"]+"'>redo</a>";
            var html = "<div><table class='tb_1'><caption>Request"+re_do_str+"</caption>";
            html += "<tr><th width='80px'>url:</th><td>" + h(req["url"]) + "</td></tr>"
            if (req["rewrite"] && req["rewrite"]["url"]) {
                html += "<tr><th>rewrite:</th><td>" + req["rewrite"]["url"] + "</td></tr>";
            }
            html += "<tr><th>proxy_urer:</th><td><b>ip:</b>" + req["client_ip"] + "&nbsp;&nbsp;<b>docid:</b>"
                    + req["@id"] + "&nbsp;<b>uname:</b>" + req["user"] + "</td></tr>";
            html += pproxy_tr_sub_table(req["form_get"], "get_params");
            html += pproxy_tr_sub_table(req["form_post"], "post_params");
            if (req["dump"]) {
                html += "<tr><th>req_dump:</th><td>" + h(Base64.decode(req["dump"])).replace(/\n/g, "<br/>")
                        + "</td></tr>";
            }
            html += "</table></div>";
            var res_link = "";
            if (res) {
                res_link = "<a href='/response?id=" + res["@id"] + "' target='_blank'>view</a>";
            }
            html += "<div><table class='tb_1'><caption>Response&nbsp;" + res_link + "</caption>"
            
            var hideBigBody=false;
            
            if (res) {
                if (res["dump"]) {
                    html += "<tr><th width='80px'>res_dump:</th><td>" + h(Base64.decode(res["dump"])).replace(/\n/g, "<br/>")
                    + "</td></tr>";
                }
                var body_str = Base64.decode(res["body"])
                var isImg = res["header"]["Content-Type"] != undefined
                        && res["header"]["Content-Type"][0].indexOf("image") > -1;

                var isStatusOk = res["status"] == 200

                var bd_json = pproxy_parseAsjson(body_str)

                if (bd_json) {
                	hideBigBody=true;
                    html += "<tr><th width='80px'>body_json:</th><td>" + bd_json + "</td></tr>";
                }
                if (isImg) {
                	hideBigBody=true;
                    html += "<tr><th>body_img:</th><td><img src='data:" + res["header"]["Content-Type"][0] + ";base64,"
                            + res["body"] + "'/></td></tr>";
                }
                if (!isImg || res["body"].length < 1000 || !isStatusOk) {
                	html += "<tr><th width='80px'>body:";
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
            $("#right_content").empty().html(html).hide().slideDown("fast")
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
            var json_str = JSON.stringify(jsonObj, null, 4)
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
    var html = "<tr><th>" + name + ":</th><td class='td_has_sub'><table class='tb_1'>";
    var i = 0;
    for ( var k in obj) {
        html += "<tr><th width='80px'>" + k + ":</th><td><ul class='td_ul'>";
        for ( var i in obj[k]) {
            html += "<li>";
            var json_str = pproxy_parseAsjson(obj[k]);
            if (json_str) {
                html += json_str;
            } else {
                html += h(obj[k])
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

function get_response(tr, docid) {
    console && console.log("get_response docid=", docid)
    var loading_msg="loading...docid=" + docid
    var isValidId=(docid+"").length>2
    if(!isValidId){
    	loading_msg="https request:no data"
    }
    $("#right_content").empty().html("<center style='margin:200px 0 auto'>"+loading_msg+"</center>")
    if(!isValidId){
    	return;
    }
    socket.emit("get_response", docid)
    $(tr).parent("tbody").find("tr").removeClass("selected")
    $(tr).addClass("selected")
    location.hash="req_"+docid
}

function bytesToString(bytes) {
    var result = "";
    for (var i = 0; i < bytes.length; i++) {
        result += String.fromCharCode(parseInt(bytes[i], 2));
    }
    return result;
}

function h(html) {
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
	
    $("#network_filter_form").change(function() {
        var form_data = $(this).serialize();
        socket.emit("client_filter", form_data);
    }).change();
    if(location.hash.match(/req_\d+/)){
        var docid=location.hash.substr(5);
        socket.emit("get_response", docid)
    }
});

function pproxy_local_save(target,id){
	if(!window.localStorage){
		return;
	}
	$(target).val(window.localStorage[id]||"").change(function(){
		window.localStorage[id]=$(this).val();
	});
}
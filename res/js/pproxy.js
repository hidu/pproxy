var socket = io.connect();
socket.on('connect', function() {
	$("#connect_status").html("<font color=green>online</font>")
	$("#network_filter_form").change();
});
socket.on("req", function(data) {
	console && console.log("req",data)
	$("#tb_network tbody").prepend(
			"<tr onclick=\"get_response(this,'" + data['docid'] + "')\">" + "<td>"
					+ data["sid"] + "</td>" + "<td></td>" + "<td>"
					+ data["host"] + "</td>" + "<td>" + data["path"] + "</td>"
					+ "</tr>")
})
socket.on("res", function(data) {
	console && console.log("res",data)
	var req = data["req"];
	var res = data["res"];
	var html = "<div><table class='tb_1'><caption>Request</caption>";
	html += "<tr><th width='80px'>url:</th><td>" + h(req["url"]) + "</td></tr>"
	if(req["rewrite"] && req["rewrite"]["url"]){
		html += "<tr><th>rewrite:</th><td>" + req["rewrite"]["url"]+"</td></tr>";
	}
	html += "<tr><th>proxy_urer:</th><td><b>ip:</b>" + req["client_ip"]+"&nbsp;&nbsp;<b>docid:</b>"+req["@id"]+ "&nbsp;<b>uname:</b>"+req["user"]+"</td></tr>";
   html+=pproxy_tr_sub_table(req["form_get"],"get_params");
   html+=pproxy_tr_sub_table(req["form_post"],"post_params");
	if(req["dump"]){
		html += "<tr><th>req_dump:</th><td><pre>" + h(Base64.decode(req["dump"]))+ "</pre></td></tr>";
	}
	html += "</table></div>";
	html += "<div><table class='tb_1'><caption>Response</caption>"

	if (res) {
		html += "<tr><th width='80px'>content-length:</th><td>"
				+ res["content_length"] + "</td></tr>"
		html += "<tr><th>status:</th><td>" + res["status"] + "</td></tr>";

		var body_str = Base64.decode(res["body"])
		var isImg = res["header"]["Content-Type"] != undefined
				&& res["header"]["Content-Type"][0].indexOf("image") > -1;

		var isStatusOk = res["status"] == 200

		if (!isImg || res["body"].length < 1000 || !isStatusOk) {
			html += "<tr><th>body:</th><td><pre>" + h(body_str)
					+ "</pre></td></tr>";
		}
		try {
			var bd_json = JSON.parse(body_str);
			if (bd_json) {
				var bd_json_str = JSON.stringify(bd_json, null, 4)
				html += "<tr><th>body_json:</th><td><pre>" + bd_json_str
						+ "</pre></td></tr>";
			}
		} catch (e) {
		}
		if (isImg) {
			html += "<tr><th>body_img:</th><td><img src='data:"
					+ res["header"]["Content-Type"][0] + ";base64,"
					+ res["body"] + "'/></td></tr>";
		}
		if(res["dump"]){
			html += "<tr><th>res_dump:</th><td><pre>"+ h(Base64.decode(res["dump"])) + "</pre></td></tr>";
		}
	}

	html += "</table></div>";
	$("#content").empty().html(html).hide().slideDown("fast")
})
socket.on("disconnect", function() {
	$("#connect_status").html("<font color=red>offline</font>")
})

function pproxy_tr_sub_table(obj,name){
	if(!obj){
		return "";
	}
	var html= "<tr><th>"+name+":</th><td><table class='tb_1'>";
	var i=0;
	for ( var k in obj) {
		html += "<tr><th width='80px'>" + k + ":</th><td>"+ h(obj[k].join("\n")) + "</td></tr>";
		i++
	}
	if(i<1){
		return "";
	}
	html += "</table></td></tr>"
	return html
}

function get_response(tr,docid) {
	console && console.log("get_response docid=", docid)
	$("#content").empty().html("<center style='margin:200px 0 auto'>loading...docid="+docid+"</center>")
	socket.emit("get_response", docid)
	$(tr).parent("tbody").find("tr").removeClass("selected")
	$(tr).addClass("selected")
}

function bytesToString(bytes) {
	var result = "";
	for (var i = 0; i < bytes.length; i++) {
		result += String.fromCharCode(parseInt(bytes[i], 2));
	}
	return result;
}

function h(str) {
	str = str.replace(/&/g, '&amp;');
	str = str.replace(/</g, '&lt;');
	str = str.replace(/>/g, '&gt;');
	str = str.replace(/'/g, '&acute;');
	str = str.replace(/"/g, '&quot;');
	str = str.replace(/\|/g, '&brvbar;');
	return str;
}

$().ready(function(){
		$("#network_filter_form").change(function(){
			var form_data=$(this).serialize();
			socket.emit("client_filter", form_data)
		}).change();
		
});
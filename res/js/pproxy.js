var socket = io.connect();
socket.on('connect', function() {
	$("#connect_status").html("<font color=green>online</font>")
});
socket.on("req", function(data) {
	$("#tb_network tbody").prepend(
			"<tr onclick=\"get_response('" + data['docid'] + "')\">" + "<td>"
					+ data["sid"] + "</td>" + "<td></td>" + "<td>"
					+ data["host"] + "</td>" + "<td>" + data["path"] + "</td>"
				   + "</tr>")
})
socket.on("res", function(data) {
	console.log(data)
	var req = data["req"]["origin"];
	var res = data["res"];
	var html = "<div><table class='tb_1'><caption>Request</caption>";
	html += "<tr><th width='100px'>url:</th><td>" + req["url"] + "</td></tr>"
	html += "<tr><th>client:</th><td>" + req["client_ip"] + "</td></tr>";
	html += "<tr><th>user:</th><td>" + req["user"] + "</td></tr>";
	for ( var k in req["header"]) {
		html += "<tr><th>" + k + ":</th><td>" + req["header"][k].join("<br/>")+"</td></tr>";
	}
	html += "<tr><th>form:</th><td><table class='tb_1'>"
	for ( var k in req["form"]) {
		html+="<tr><th width='100px'>"+k+":</th><td>"+req["form"][k].join("<br/>")+"</td></tr>";
	}
	html +="</table></td></tr>"
		
	html += "</table></div>";
	html += "<div><table class='tb_1'><caption>Response</caption>"
 if(res){
	html += "<tr><th width='100px'>content-length:</th><td>"+res["content_length"]+"</td></tr>"
	html += "<tr><th>status:</th><td>" + res["status"] + "</td></tr>";
	html += "<tr><th>body:</th><td>" + res["body"] + "</td></tr>";
	try{
		var bd_json=JSON.parse(res["body"]);
		if(bd_json){
			var bd_json_str=JSON.stringify(bd_json,null,4)
			html += "<tr><th>body_json:</th><td><pre>" +bd_json_str + "</pre></td></tr>";
		}
   }catch(e){}
   for ( var k in res["header"]) {
		html += "<tr><th>" + k + ":</th><td>" + res["header"][k].join("<br/>")+ "</td></tr>";
	}
 }
   
	html += "</table></div>";
	$("#content").html(html)
})
socket.on("disconnect", function() {
	$("#connect_status").html("<font color=red>offline</font>")
})

function get_response(docid) {
	console.log("get_response docid=", docid)
	socket.emit("get_response", docid)
}

function bytesToString(bytes) {
	var result = "";
	for (var i = 0; i < bytes.length; i++) {
		result += String.fromCharCode(parseInt(bytes[i], 2));
	}
	return result;
}
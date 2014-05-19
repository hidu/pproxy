var socket = io.connect();
socket.on('connect', function() {
	$("#connect_status").html("<font color=green>online</font>")
});
socket.on("req", function(data) {
	$("#tb_network tbody").prepend("<tr onclick=\"get_response('"+data['docid']+"')\">" +
											"<td>"+data["sid"]+"</td>" +
											"<td></td>" +
											"<td>"+data["host"]+"</td>" +
										   "<td>"+data["path"]+"</td>" +
										   "<td></td>" +
										   "<td></td>" +
										   "</tr>")
})
socket.on("res", function(data) {
	console.log(data)
	var req=data["req"];
	var res=data["res"];
	var html="<div><table class='tb_1'><caption>Request</caption>";
	html+="<tr><th width='100px'>url:</th><td>"+req["url"]+"</td></tr>"
	html+="<tr><th>client:</th><td>"+req["client_ip"]+"</td></tr>";
	html+="<tr><th>user:</th><td>"+req["user"]+"</td></tr>";
	for(var k in req["header"]){
		html+="<tr><th>"+k+":</th><td>"+req["header"][k].join("&nbsp;")+"</td></tr>";
	}
	html+="</table></div>";
	html+="<div><table class='tb_1'><caption>Response</caption>"
	html+="<tr><th width='100px'>content-length:</th><td></td></tr>"
	html+="<tr><th>body:</th><td>"+res["body"]+"</td></tr>";
	html+="</table></div>";
	$("#content").html(html)
})
socket.on("disconnect", function() {
	$("#connect_status").html("<font color=red>offline</font>")
})

function get_response(docid){
	console.log("get_response docid=",docid)
	socket.emit("get_response",docid)
}


function bytesToString(bytes) {
  var result = "";
  for (var i = 0; i < bytes.length; i++) {
    result += String.fromCharCode(parseInt(bytes[i], 2));
  }
  return result;
}
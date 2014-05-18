var socket = io.connect();
socket.on('connect', function() {
	$("#connect_status").html("<font color=green>online</font>")
});
socket.on("req", function(data) {
	$("#tb_network tbody").prepend("<tr onclick='get_response("+data['docid']+")'>" +
											"<td>"+data["sid"]+"</td>" +
											"<td></td>" +
											"<td>"+data["host"]+"</td>" +
										   "<td>"+data["path"]+"</td>" +
										   "<td></td>" +
										   "<td></td>" +
										   "</tr>")
})
socket.on("res", function(data) {
	$("#content").text(data["body"])
})
socket.on("disconnect", function() {
	$("#connect_status").html("<font color=red>offline</font>")
})

function get_response(docid){
	socket.emit("get_response",docid)
}


function bytesToString(bytes) {
  var result = "";
  for (var i = 0; i < bytes.length; i++) {
    result += String.fromCharCode(parseInt(bytes[i], 2));
  }
  return result;
}
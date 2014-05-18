var socket = io.connect();
socket.on('connect', function() {
	$("#connect_status").html("<font color=green>online</font>")
});
socket.on("req", function(sid,host,path) {
	$("#tb_network tbody").append("<tr><td>"+sid+"</td><td>"+host+"</td><td>"+path+"</td><td></td><td></td></tr>")
})
socket.on("disconnect", function() {
	$("#connect_status").html("<font color=red>offline</font>")
})
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
	$("#content").empty().html("loading...")
	// var req = JSON.parse(Base64.decode(data["req"]["origin"]));
	var req = data["req"];
	var res = data["res"];
	var html = "<div><table class='tb_1'><caption>Request</caption>";
	html += "<tr><th width='80px'>url:</th><td>" + h(req["url"]) + "</td></tr>"
	html += "<tr><th>client:</th><td>" + req["client_ip"]+"&nbsp;&nbsp;docid:"+req["@id"]+ "</td></tr>";
	html += "<tr><th>user:</th><td>" + req["user"] + "</td></tr>";

	// for ( var k in req["header"]) {
	// html += "<tr><th>" + k + ":</th><td>" +
	// req["header"][k].join("<br/>")+"</td></tr>";
	// }
	if (req["form"]) {
		html += "<tr><th>form:</th><td><table class='tb_1'>"
		for ( var k in req["form"]) {
			html += "<tr><th width='80px'>" + k + ":</th><td>"
					+ req["form"][k].join("<br/>") + "</td></tr>";
		}
		html += "</table></td></tr>"
	}
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
		for ( var k in res["header"]) {
			html += "<tr><th>" + k + ":</th><td>"
					+ res["header"][k].join("<br/>") + "</td></tr>";
		}
		if(res["dump"]){
			html += "<tr><th>res_dump:</th><td><pre>"+ h(Base64.decode(res["dump"])) + "</pre></td></tr>";
		}
	}

	html += "</table></div>";
	$("#content").empty().html(html)
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

function h(str) {
	str = str.replace(/&/g, '&amp;');
	str = str.replace(/</g, '&lt;');
	str = str.replace(/>/g, '&gt;');
	str = str.replace(/'/g, '&acute;');
	str = str.replace(/"/g, '&quot;');
	str = str.replace(/\|/g, '&brvbar;');
	return str;
}
var socket = io.connect();
socket.on('connect', function() {
    $("#connect_status").html("<font>online</font>")
    $("#network_filter_form").change();
});
socket.on("req", function(data) {
    console && console.log("req", data)
    $("#tb_network tbody").prepend(
            "<tr onclick=\"get_response(this,'" + data['docid'] + "')\">" + "<td>" + data["sid"] + "</td>"
                    + "<td></td>" + "<td>" + data["host"] + "</td>" + "<td>" + data["path"] + "</td>" + "</tr>")
})
socket.on("res",
        function(data) {
            console && console.log("res", data)
            var req = data["req"];
            var res = data["res"];
            var html = "<div><table class='tb_1'><caption>Request</caption>";
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

            if (res) {
                html += "<tr><th width='80px'>status:</th><td>" + res["status"] + "</td></tr>";

                var body_str = Base64.decode(res["body"])
                var isImg = res["header"]["Content-Type"] != undefined
                        && res["header"]["Content-Type"][0].indexOf("image") > -1;

                var isStatusOk = res["status"] == 200

                var bd_json = pproxy_parseAsjson(body_str)

                if (!isImg || res["body"].length < 1000 || !isStatusOk) {
                    html += "<tr><th>body:</th><td>" + h(body_str).replace(/\n/g, "<br/>") + "</td></tr>";
                }
                if (bd_json) {
                    html += "<tr><th>body_json:</th><td>" + bd_json + "</td></tr>";
                }
                if (isImg) {
                    html += "<tr><th>body_img:</th><td><img src='data:" + res["header"]["Content-Type"][0] + ";base64,"
                            + res["body"] + "'/></td></tr>";
                }
                if (res["dump"]) {
                    html += "<tr><th>res_dump:</th><td>" + h(Base64.decode(res["dump"])).replace(/\n/g, "<br/>")
                            + "</td></tr>";
                }
            }

            html += "</table></div>";
            $("#content").empty().html(html).hide().slideDown("fast")
        })
socket.on("disconnect", function() {
    $("#connect_status").html("<font color=red>offline</font>")
})

function pproxy_parseAsjson(str) {
    try {
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
    var html = "<tr><th>" + name + ":</th><td><table class='tb_1'>";
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
    $("#content").empty().html("<center style='margin:200px 0 auto'>loading...docid=" + docid + "</center>")
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
    $("#network_filter_form").change(function() {
        var form_data = $(this).serialize();
        socket.emit("client_filter", form_data)
    }).change();

});
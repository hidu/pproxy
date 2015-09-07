package utils

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	//	"fmt"
)

func Net_isLocalIp(host string) bool {
	ips, _ := net.LookupIP(host)
	for _, ip := range ips {
		if ip.IsLoopback() {
			return true
		}
	}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			_, ip_g, err := net.ParseCIDR(addr.String())
			if err == nil {
				for _, ip := range ips {
					if ip_g.Contains(ip) {
						return true
					}
				}
			}
		}
	}
	return false
}

func Net_getHostPortFromReq(req *http.Request) (host string, port int, err error) {
	urlStr := ""
	if req.URL.Scheme != "" {
		urlStr = req.URL.Scheme + "://" + req.Host
	} else {
		urlStr = "http://" + req.Host
	}
	return Net_getHostPortFromUrl(urlStr)
}

func Net_getHostPortFromUrl(urlStr string) (host string, port int, err error) {
	urlObj, err := url.Parse(urlStr)
	if err != nil {
		return "", 0, err
	}
	host, port, err = parseHostPort(urlObj.Host)
	if err == nil && port == 0 {
		switch urlObj.Scheme {
		case "http":
		case "ws":
			port = 80
			break
		case "https":
		case "wss":
			port = 443
			break
		default:
			break
		}
	}
	return
}

func parseHostPort(hostPortstr string) (host string, port int, err error) {
	var port_str string
	if !strings.Contains(hostPortstr, ":") {
		hostPortstr += ":0"
	}
	host, port_str, err = net.SplitHostPort(hostPortstr)
	if err != nil {
		return
	}
	port, err = strconv.Atoi(port_str)
	if err != nil {
		return
	}
	return
}

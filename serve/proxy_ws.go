package serve

/**
@see
* https://github.com/koding/websocketproxy/blob/master/websocketproxy.go
*/

import (
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

var (
	// DefaultUpgrader specifies the parameters for upgrading an HTTP
	// connection to a WebSocket connection.
	DefaultUpgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

// WebsocketProxy is an HTTP Handler that takes an incoming WebSocket
// connection and proxies it to another server.
type WebsocketProxy struct {
	// Upgrader specifies the parameters for upgrading a incoming HTTP
	// connection to a WebSocket connection. If nil, DefaultUpgrader is used.
	Upgrader *websocket.Upgrader
	// Dialer contains options for connecting to the backend WebSocket server.
	// If nil, DefaultDialer is used.
	ser *ProxyServe
}

// NewProxy returns a new Websocket reverse proxy that rewrites the
// URL's to the scheme, host and base path provider in target.
func NewWsProxy(ser *ProxyServe) *WebsocketProxy {
	return &WebsocketProxy{ser: ser}
}

var wsProxyIgnoreHeader map[string]int = map[string]int{
	"upgrade":               1,
	"connection":            1,
	"sec-websocket-version": 1,
	"sec-websocket-key":     1,
	"host":                  1,
}

// ServeHTTP implements the http.Handler that proxies WebSocket connections.
func (w *WebsocketProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Connect to the backend URL, also pass the headers we get from the requst
	// together with the Forwarded headers we prepared above.
	// TODO: support multiplexing on the same backend connection instead of
	// opening a new TCP connection time for each request. This should be
	// optional:
	// http://tools.ietf.org/html/draft-ietf-hybi-websocket-multiplexing-01
	req.URL.Scheme = "ws" + req.URL.Scheme[4:]

	removeHeader(req)
	reqCtx := NewRequestCtx(w.ser, req)
	w.ser.regirestReq(req, reqCtx)

	rewrite_code := w.ser.reqRewrite(req, reqCtx)
	reqCtx.HasBroadcast = w.ser.Broadcast_Req(req, reqCtx)
	hasSave := false

	saveReq := func() {
		if hasSave {
			return
		}
		w.ser.saveRequestData(req, reqCtx)
		reqCtx.PrintLog()
		hasSave = true
	}
	showErrorRes := func(msg string) {
		reqCtx.Msg = msg
		log.Println(msg)
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(msg))
	}

	defer saveReq()
	if rewrite_code != 200 && rewrite_code != 304 {
		showErrorRes("websocket rewrite failed")
		return
	}
	hasOrigin := req.Header.Get("Origin") != ""
	if rewrite_code == 200 && hasOrigin {
		req.Header.Set("Origin", "http://"+req.Host)
	}
	requestHeader := http.Header{}
	for k, v := range req.Header {
		if _, has := wsProxyIgnoreHeader[strings.ToLower(k)]; !has {
			requestHeader[k] = v
		}
	}

	if w.ser.Debug {
		req_dump_debug, _ := httputil.DumpRequest(req, true)
		log.Println("rewrite_code:\n", rewrite_code)
		log.Println("ws_req_after_rewrite:\n", string(req_dump_debug), "\n")
		log.Println("ws_requestHeader:", requestHeader, "\n")
	}
	_url := req.URL.String()
	connBackend, resp, err := getWsDialer(req, requestHeader)
	if err != nil {
		_logMsg := fmt.Sprintf("websocketproxy: couldn't dial to remote backend url %s,url:%s\n", err, _url)
		showErrorRes(_logMsg)
		return
	}

	saveReq()

	defer connBackend.Close()
	upgrader := w.Upgrader
	if w.Upgrader == nil {
		upgrader = DefaultUpgrader
	}
	// Only pass those headers to the upgrader.
	upgradeHeader := http.Header{}
	upgradeHeader.Set("Sec-WebSocket-Protocol", resp.Header.Get(http.CanonicalHeaderKey("Sec-WebSocket-Protocol")))
	upgradeHeader.Set("Set-Cookie", resp.Header.Get(http.CanonicalHeaderKey("Set-Cookie")))
	// Now upgrade the existing incoming request to a WebSocket connection.
	// Also pass the header that we gathered from the Dial handshake.
	connPub, err := upgrader.Upgrade(rw, req, upgradeHeader)
	if err != nil {
		log.Printf("websocketproxy: couldn't upgrade %s\n", err)
		showErrorRes("websocket error")
		return
	}
	defer connPub.Close()
	errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errc <- err
	}
	// Start our proxy now, everything is ready...
	go cp(connBackend.UnderlyingConn(), connPub.UnderlyingConn())
	go cp(connPub.UnderlyingConn(), connBackend.UnderlyingConn())
	<-errc
}

func getWsDialer(req *http.Request, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {

	d := &websocket.Dialer{}

	var deadline time.Time
	if d.HandshakeTimeout != 0 {
		deadline = time.Now().Add(d.HandshakeTimeout)
	}

	netDial := d.NetDial
	if netDial == nil {
		netDialer := &net.Dialer{Deadline: deadline}
		netDial = netDialer.Dial
	}
	_host, _port, _ := parseHostPort(req.URL.Host)
	if _port == 0 {
		switch req.URL.Scheme {
		case "ws":
			_port = 80
			break
		case "wss":
			_port = 443
			break
		default:
			break
		}
	}

	netConn, err := netDial("tcp", fmt.Sprintf("%s:%d", _host, _port))
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if netConn != nil {
			netConn.Close()
		}
	}()

	if err := netConn.SetDeadline(deadline); err != nil {
		return nil, nil, err
	}

	if req.URL.Scheme == "wss" {
		_reqHost, _, _ := parseHostPort(req.Host)
		cfg := d.TLSClientConfig
		if cfg == nil {
			cfg = &tls.Config{ServerName: _reqHost}
		} else if cfg.ServerName == "" {
			shallowCopy := *cfg
			cfg = &shallowCopy
			cfg.ServerName = _reqHost
		}
		tlsConn := tls.Client(netConn, cfg)
		netConn = tlsConn
		if err := tlsConn.Handshake(); err != nil {
			return nil, nil, err
		}
		if !cfg.InsecureSkipVerify {
			if err := tlsConn.VerifyHostname(cfg.ServerName); err != nil {
				return nil, nil, err
			}
		}
	}

	readBufferSize := d.ReadBufferSize
	if readBufferSize == 0 {
		readBufferSize = 4096
	}

	writeBufferSize := d.WriteBufferSize
	if writeBufferSize == 0 {
		writeBufferSize = 4096
	}

	if len(d.Subprotocols) > 0 {
		h := http.Header{}
		for k, v := range requestHeader {
			h[k] = v
		}
		h.Set("Sec-Websocket-Protocol", strings.Join(d.Subprotocols, ", "))
		requestHeader = h
	}
	conn, resp, err := websocket.NewClient(netConn, req.URL, requestHeader, readBufferSize, writeBufferSize)
	if err != nil {
		return nil, resp, err
	}

	netConn.SetDeadline(time.Time{})
	netConn = nil // to avoid close in defer.
	return conn, resp, nil
}

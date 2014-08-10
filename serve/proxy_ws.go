package serve

/**
@see
* https://github.com/koding/websocketproxy/blob/master/websocketproxy.go
*/

import (
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
)

var (
	// DefaultUpgrader specifies the parameters for upgrading an HTTP
	// connection to a WebSocket connection.
	DefaultUpgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	// DefaultDialer is a dialer with all fields set to the default zero values.
	DefaultDialer = websocket.DefaultDialer
)

// WebsocketProxy is an HTTP Handler that takes an incoming WebSocket
// connection and proxies it to another server.
type WebsocketProxy struct {
	// Upgrader specifies the parameters for upgrading a incoming HTTP
	// connection to a WebSocket connection. If nil, DefaultUpgrader is used.
	Upgrader *websocket.Upgrader
	// Dialer contains options for connecting to the backend WebSocket server.
	// If nil, DefaultDialer is used.
	Dialer *websocket.Dialer
}

// ProxyHandler returns a new http.Handler interface that reverse proxies the
// request to the given target.
func ProxyHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		NewWsProxy().ServeHTTP(rw, req)
	})
}

// NewProxy returns a new Websocket reverse proxy that rewrites the
// URL's to the scheme, host and base path provider in target.
func NewWsProxy() *WebsocketProxy {
	return &WebsocketProxy{}
}

// ServeHTTP implements the http.Handler that proxies WebSocket connections.
func (w *WebsocketProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	dialer := w.Dialer
	if w.Dialer == nil {
		dialer = DefaultDialer
	}
	// Connect to the backend URL, also pass the headers we get from the requst
	// together with the Forwarded headers we prepared above.
	// TODO: support multiplexing on the same backend connection instead of
	// opening a new TCP connection time for each request. This should be
	// optional:
	// http://tools.ietf.org/html/draft-ietf-hybi-websocket-multiplexing-01
	req.URL.Scheme = "ws" + req.URL.Scheme[4:]
	_url := req.URL.String()
	connBackend, resp, err := dialer.Dial(_url, req.Header)
	if err != nil {
		log.Printf("websocketproxy: couldn't dial to remote backend url %s,url:%s\n", err, _url)
		return
	}
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

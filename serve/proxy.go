package serve

import (
	"fmt"
	"github.com/hidu/goproxy"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

//
type HttpProxy struct {
	GoProxy            *goproxy.ProxyHttpServer
	ser                *ProxyServe
	ctxs               map[string]*requestCtx
	mu                 sync.RWMutex
	goproxyMitmConnect *goproxy.ConnectAction
}

func NewHttpProxy(ser *ProxyServe) *HttpProxy {
	proxy := new(HttpProxy)
	proxy.ser = ser
	proxy.GoProxy = goproxy.NewProxyHttpServer()
	tr := ser.conf.getTransport()
	if tr != nil {
		proxy.GoProxy.Tr = tr
	}
	proxy.ctxs = make(map[string]*requestCtx)
	if proxy.ser.conf.SslOn {
		proxy.goproxyMitmConnect = &goproxy.ConnectAction{
			Action:    goproxy.ConnectMitm,
			TLSConfig: goproxy.TLSConfigFromCA(&proxy.ser.conf.SslCert),
		}
		proxy.GoProxy.OnRequest().HandleConnectFunc(proxy.httpsHandle)
	}
	proxy.GoProxy.OnRequest().DoFunc(my_requestHanderFunc)
	proxy.GoProxy.OnResponse().DoFunc(proxy.onResponse)
	return proxy
}

func my_requestHanderFunc(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	log.Println("trace_my_requestHanderFunc call url:", r.URL.String())
	return r, nil
}

const PROXY_CTX_NAME = "X-PPROXY-CTX-ID"

func (proxy *HttpProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	//	fmt.Println("call url:",req.URL.String())
	proxy.GoProxy.ServeHTTP(rw, req)
}

func (proxy *HttpProxy) httpsHandle(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	log.Println("https conn", host, ctx.Req.URL.String())
	return proxy.goproxyMitmConnect, host
}

func (proxy *HttpProxy) RoundTrip(ctx *requestCtx) {
	sid := fmt.Sprintf("%d", ctx.SessionID)
	ctx.Req.Header.Set(PROXY_CTX_NAME, sid)
	func() {
		proxy.mu.Lock()
		defer proxy.mu.Unlock()
		proxy.ctxs[sid] = ctx
	}()

	defer func() {
		proxy.mu.Lock()
		defer proxy.mu.Unlock()
		if _, has := proxy.ctxs[sid]; has {
			delete(proxy.ctxs, sid)
		}
	}()

	if ctx.Req.Header.Get("Upgrade") != "" {
		proxy.roundTripUpgrade(ctx)
		return
	}
	proxy.ServeHTTP(ctx.Rw, ctx.Req)
}

func (proxy *HttpProxy) getReqCtx(req *http.Request) *requestCtx {
	sid := req.Header.Get(PROXY_CTX_NAME)
	if sid == "" {
		return nil
	}
	proxy.mu.RLock()
	defer proxy.mu.RUnlock()
	if ctx, has := proxy.ctxs[sid]; has {
		return ctx
	}
	return nil
}

func (proxy *HttpProxy) onResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil || resp.Request == nil {
		return resp
	}
	reqCtx := proxy.getReqCtx(resp.Request)

	if reqCtx != nil {
		reqCtx.saveResponse(resp)
	}
	return resp
}

func (proxy *HttpProxy) roundTripUpgrade(ctx *requestCtx) (err error) {
	//save it,so we know it has been closed
	defer func() {
		resp := &http.Response{
			Request: ctx.Req,
			Header:  make(http.Header),
			Body:    nil,
		}
		ctx.saveResponse(resp)
	}()

	reqDump, err := httputil.DumpRequest(ctx.Req, false)
	if err != nil {
		ctx.Msg = "dump req failed:" + err.Error()
		return
	}
	ctx.SetTimePoint("startDial")
	dia, err := net.Dial("tcp", ctx.DestAddr())
	if err != nil {
		ctx.Msg = "dia connect " + ctx.DestAddr() + " failed!" + err.Error()
		return
	}
	defer dia.Close()
	_, err = dia.Write(reqDump)
	if err != nil {
		return
	}

	hijack, _ := ctx.Rw.(http.Hijacker)
	conn, _, _ := hijack.Hijack()

	errc := make(chan error, 2)

	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errc <- err

		time.AfterFunc(3*time.Second, func() {
			dia.Close()
			conn.Close()
		})
	}

	go cp(dia, conn)
	go cp(conn, dia)
	<-errc
	return
}

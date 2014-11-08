package serve

import (
	"io"
	"net"
	"fmt"
	"net/http"
	"net/http/httputil"
)

type proxyRoundTripper func(*requestCtx) (*http.Response, error)

func RoundTrip_Upgrade(ctx *requestCtx) (resp *http.Response,err error) {
	req_dump, _ := httputil.DumpRequest(ctx.Req, false)

	dia, err := net.Dial("tcp", ctx.DestAddr())
	if err != nil {
		fmt.Println("upgrade proxy failed:", err)
		ctx.Rw.WriteHeader(http.StatusBadGateway)
		ctx.Rw.Write([]byte("error"))
		return
	}

	dia.Write(req_dump)

	errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		fmt.Println("cp err:", err)
		errc <- err
	}

	hijack, _ := ctx.Rw.(http.Hijacker)
	conn, _, _ := hijack.Hijack()

	go cp(conn, dia)
	go cp(dia, conn)
	<-errc
	return
}

func RoundTrip_Default(ctx *requestCtx) (*http.Response, error) {
	return ctx.Tr.RoundTrip(ctx.Req)
}

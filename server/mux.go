package main

import (
	"context"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/google/tcpproxy"
	"github.com/inconshreveable/muxado"
)

func StartMuxadoSession(port int, rwc io.ReadWriteCloser, done <-chan bool) error {
	mux := muxado.Client(rwc, nil)

	var p tcpproxy.Proxy

	muxDest := &tcpproxy.DialProxy{
		Addr: "127.0.0.1:22",
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return mux.Open()
		},
	}
	listen := "127.0.0.1:" + strconv.Itoa(port)
	log.Printf("Listening on %s", listen)
	p.AddRoute(listen, muxDest)

	go func() {
		<-done
		p.Close()
	}()

	return p.Run()
}

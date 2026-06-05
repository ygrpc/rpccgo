//go:build cgo

package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	connect "connectrpc.com/connect"
	backend "example.com/rpccgo-connect/internal/backend"
	greeterv1 "example.com/rpccgo-connect/proto"
	"golang.org/x/net/http2"
)

func init() {
	if baseURL := strings.TrimRight(os.Getenv("RPCCGO_CONNECT_URL"), "/"); baseURL != "" {
		client := greeterv1.NewGreeterClient(h2cClient(), baseURL)
		if err := greeterv1.RegisterGreeterConnectRemoteServer(client); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		log.Fatal(err)
	}
}

func h2cClient() connect.HTTPClient {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}
}

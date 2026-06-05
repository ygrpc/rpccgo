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
	if baseURL := strings.TrimRight(argValue("--connect-url"), "/"); baseURL != "" {
		client := greeterv1.NewGreeterClient(h2cClient(), baseURL)
		if err := greeterv1.RegisterGreeterConnectRemoteServer(client); err != nil {
			log.Fatal(err)
		}
		return
	}
	if argValue("--server") == "connect_handler" {
		if err := greeterv1.RegisterGreeterConnectHandler(backend.ConnectGreeter{}); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		log.Fatal(err)
	}
}

func argValue(name string) string {
	prefix := name + "="
	for index, arg := range os.Args[1:] {
		if arg == name && index+2 <= len(os.Args[1:]) {
			return os.Args[index+2]
		}
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return ""
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

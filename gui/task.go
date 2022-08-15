package main

import (
	"context"
	"fmt"

	"happo31.jp/th123tunnel/proxy"
)

func StartClient(context context.Context, localAddr string, remoteAddr string) (chan error, error) {
	p, err := proxy.New(localAddr)
	if err != nil {
		return nil, err
	}
	fmt.Printf("start client %s\n", remoteAddr)
	return p.StartClient(context, remoteAddr)
}

func StartServer(context context.Context, localAddr string, serverPort int) (chan error, error) {
	p, err := proxy.New(localAddr)
	if err != nil {
		return nil, err
	}
	fmt.Printf("start server %d\n", serverPort)
	return p.StartServer(context, serverPort)
}

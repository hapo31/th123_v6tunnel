package main

import (
	"fmt"
	"log"
	"net"

	"github.com/mattn/go-tty"
	"happo31.jp/th123tunnel/proxy"
)

func main() {

	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}

	defer tty.Close()

	recvAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 10800,
	}

	proxy := proxy.Proxy{}

	
	go proxy.StartLocalServer(recvAddr)
	defer proxy.StopReceive()
	fmt.Println("started server. (E key pressed to stop.)")

	for {
		r, err := tty.ReadRune()

		if err != nil {
			log.Fatal(err)
		}

		if string(r) == "e" {
			break
		}

		go func() {
			for {
				msg, ok := <-proxy.RecvChannel
				if !ok {
					log.Fatal("Faild recv message")
					break
				}
				fmt.Printf("[%v]\n", msg.Addr)
			}
		}()
	}
	fmt.Println("bye.")
}

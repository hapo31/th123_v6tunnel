package main

import (
	"fmt"
	"log"
	"os"

	"happo31.jp/th123tunnel/proxy"
)

func main() {

	p, err := proxy.New(10800)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	chann, err := p.StartClient("127.0.0.1:9900")

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	for {
		<- chann
		fmt.Println("process end.")
		break
	}
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mattn/go-tty"
	"happo31.jp/th123tunnel/proxy"
)

func main() {

	var (
		s          = flag.Bool("s", false, "server mode")
		c          = flag.Bool("c", false, "client mode")
		th123Port  = flag.Int("th", 10800, "th123 port")
		serverAddr = flag.String("i", "INVALID_ADDRESS", "proxy server ip address")
	)

	flag.Parse()

	if *s && *c {
		fmt.Println("Do not both -s -c flag.")
		os.Exit(1)
	}

	if *c && serverAddr == nil {
		fmt.Println("Must set be -i flag in client mode.")
		os.Exit(1)
	}

	p, err := proxy.New(*th123Port)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	var abortChan chan bool

	if *c {
		fmt.Printf("mode: Client(use th123 port:%d)\n", *th123Port)

		abortChan, err = p.StartClient(*serverAddr)

		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	} else if *s {
		fmt.Printf("mode: Server(use th123 port:%d)\n", *th123Port)
		abortChan, err = p.StartServer(*th123Port)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}

	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}

	defer tty.Close()

	runeChan := make(chan string)

	go func() {
		for {
			r, err := tty.ReadRune()
			if err != nil {
				log.Fatal(err)
			}
			runeChan <- string(r)
		}
	}()

loop:
	for {
		select {
		case <-abortChan:
			break loop
		case str := <-runeChan:
			if str == "q" {
				break loop
			}
		}
	}

	fmt.Println("bye.")
}

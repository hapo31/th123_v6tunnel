package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/mattn/go-tty"
	"happo31.jp/th123tunnel/proxy"
)

func main() {

	var (
		t          = flag.Bool("t", false, "test mode")
		s          = flag.Bool("s", false, "server mode")
		c          = flag.Bool("c", false, "client mode")
		ipv6       = flag.Bool("6", false, "ipv6 mode")
		th123Port  = flag.Int("th", 10800, "th123 port")
		th123Addr  = flag.String("th_addr", fmt.Sprintf("127.0.0.1:%d", *th123Port), "th123 addr")
		serverPort = flag.Int("p", *th123Port+1, "server port")
		remoteAddr = flag.String("i", "", "remote ip address")
	)

	flag.Parse()

	clientMode := *c || *remoteAddr != ""

	if *t {
		c, _ := net.Dial("udp", *remoteAddr)
		c.Write([]byte("hello world"))
		os.Exit(0)
	}

	if *s && *c {
		fmt.Println("Do not both -s -c flag.")
		os.Exit(1)
	}

	if *c && len(*remoteAddr) == 0 {
		fmt.Println("Must set be -i flag in client mode.")
		os.Exit(1)
	}

	p, err := proxy.New(*th123Addr)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	var abortChan chan bool
	var errChan chan error

	if clientMode {
		fmt.Printf("mode: Client(use th123 port:%d)\n", *th123Port)

		errChan, err = p.StartClient(*remoteAddr)

		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	} else if *s || !*s && !*c {
		// フラグが指定されなかった場合はサーバーモードで起動
		fmt.Printf("mode: Server(use th123 port:%d)\n", *th123Port)
		errChan, err = p.StartServer(*serverPort, *ipv6)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	} else {
		println("Unexcepted options.")
		os.Exit(1)

	}

	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}

	defer tty.Close()

	runeChan := make(chan string)
	waitChan := make(chan bool)

	go func() {
		println("> (type q was quit.)")
		for {
			r, err := tty.ReadRune()
			if err != nil {
				log.Fatal(err)
			}
			runeChan <- string(r)
			<-waitChan
		}
	}()

loop:
	for {
		select {
		case <-abortChan:
			break loop
		case err := <-errChan:
			log.Fatal(err)
			os.Exit(1)
		case str := <-runeChan:
			if str == "q" {
				break loop
			}
		}
		waitChan <- true
	}

	fmt.Println("bye.")
}

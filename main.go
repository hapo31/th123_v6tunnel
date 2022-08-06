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
		th123Port  = flag.Int("th", 10800, "th123 port")
		serverAddr = flag.String("i", "", "server ip address")
	)

	flag.Parse()

	if *t {
		c, _ := net.Dial("udp", *serverAddr)
		c.Write([]byte("hello world"))
		os.Exit(0)
	}

	if *s && *c {
		fmt.Println("Do not both -s -c flag.")
		os.Exit(1)
	}

	if *c && len(*serverAddr) == 0 {
		fmt.Println("Must set be -i flag in client mode.")
		os.Exit(1)
	}

	p, err := proxy.New(*th123Port)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	var abortChan chan bool
	var errChan chan error

	if *c {
		fmt.Printf("mode: Client(use th123 port:%d)\n", *th123Port)

		abortChan, errChan, err = p.StartClient(*serverAddr)

		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	} else if *s || !*s && !*c {
		// フラグが指定されなかった場合はサーバーモードで起動
		fmt.Printf("mode: Server(use th123 port:%d)\n", *th123Port)
		abortChan, errChan, err = p.StartServer(*th123Port)
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
	waitChan := make(chan bool)

	go func() {
		for {
			println("> (type q was quit.)")
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

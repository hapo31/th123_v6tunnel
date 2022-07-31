package main

import (
	"fmt"
	"net"
)

func main() {
	recvAddr := &net.UDPAddr {
		IP: net.ParseIP("127.0.0.1"),
		Port: 10800,
	}

	recvlistner, err := net.ListenUDP("udp", recvAddr)
	if err != nil {
		fmt.Printf("%s", err.Error())
	}

	var sendAddr *net.UDPAddr
	var sendConn net.Conn
	
	fmt.Printf("starting...")
	for {
		recvBuf := make([]byte, 64)
		n, addr, err := recvlistner.ReadFromUDP(recvBuf)
		if err != nil {
			fmt.Printf("%s", err.Error())
		}

		go func() {
			sendBuf := make([]byte, 64)
			copy(sendBuf, recvBuf)
			if sendConn == nil {
				sendAddr = &net.UDPAddr {
					IP: addr.IP,
					Port: 10801,
				}
				sendConn, err = net.Dial("udp", sendAddr.String())
				if err != nil {
					fmt.Printf("%s", err.Error())
				}
			}
			if sendConn != nil {
				sendConn.Write(sendBuf)
			}
			fmt.Printf("[%v]", addr.String())
			for i := 0; i < n; i++ {
				fmt.Printf("%02x,", sendBuf[i])
			}
			fmt.Println("")
		}()

		go func() {
			if (sendAddr != nil) {
				for {

				}
			}
		}()
	}

}

// func passthrough(addr string, port int) {
// 	recvAddr := &net.UDPAddr {
// 		IP: net.ParseIP(addr),
// 		Port: port,
// 	}
// 	recvlistner, err := net.ListenUDP("udp", recvAddr)
// 	if err != nil {
// 		fmt.Printf("%s", err.Error())
// 		os.Exit(1)
// 	}
// }
package proxy

import (
	"fmt"
	"log"
	"net"
	"time"
)

type Error int

type Mode int

const BUFFER_SIZE = 128

const (
	Client Mode = iota
	Server
)

const (
	Success Error = iota
	Runtime
	TimedOut
	NotArrivedData
)

type ProxyChannel struct {
	Addr *net.UDPAddr
	Buf  []byte
}

type Proxy struct {
	LocalAddr *net.UDPAddr
}

func New(th123Port int) (Proxy, error) {
	p := Proxy{}
	p.LocalAddr = &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: th123Port,
	}

	return p, nil
}

func (p *Proxy) StartClient(sendAddrStr string) (chan error, error) {

	errChan := make(chan error)
	remoteAddr, err := net.ResolveUDPAddr("udp", sendAddrStr)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	// 天則クライアントの待ち受け及びリモートからの通信待ち受け
	go func() {
		for {
			localConn, err := net.ListenUDP("udp", p.LocalAddr)
			if err != nil {
				log.Fatal(err)
				errChan <- err
				return
			}

			remoteConn, err := net.DialUDP("udp", nil, remoteAddr)
			if err != nil {
				log.Fatal(err)
				errChan <- err
				return
			}

			code, err := pass(localConn, remoteConn, nil, func(r *net.UDPAddr) bool {
				fmt.Printf("received:%s, me: %s\n", r.String(), remoteConn.LocalAddr().String())
				go pass(remoteConn, localConn, r, func(rr *net.UDPAddr) bool {
					// TODO: ここが出ない
					fmt.Printf("response:%s\n", rr.String())
					return true
				})
				return true
			})

			remoteConn.Close()
			localConn.Close()

			if err != nil {
				errChan <- err
				return
			}
			if code != Success {
				if code == TimedOut {
					continue
				}
			}
		}
	}()

	return errChan, nil

}

func (p *Proxy) StartServer(proxyPort int, ipv6 bool) (chan error, error) {
	var addr string
	if ipv6 {
		addr = fmt.Sprintf("[::]:%d", proxyPort)
	} else {
		addr = fmt.Sprintf("0.0.0.0:%d", proxyPort)
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	fmt.Printf("wait from in %s\n", addr)

	errChan := make(chan error)
	// リモートからの通信待ち受け
	go func() {
		for {
			remoteConn, err := net.ListenPacket("udp", remoteAddr.String())
			if err != nil {
				log.Fatal(err)
				errChan <- err
				return
			}

			localConn, err := net.Dial("udp", p.LocalAddr.String())
			if err != nil {
				log.Fatal(err)
				errChan <- err
				return
			}

			code, err := pass(remoteConn.(*net.UDPConn), localConn.(*net.UDPConn), nil, func(r *net.UDPAddr) bool {
				fmt.Printf("accepted:%s\n", r.AddrPort().String())
				go pass(localConn.(*net.UDPConn), remoteConn.(*net.UDPConn), r, func(_ *net.UDPAddr) bool { return true })
				return true
			})

			remoteConn.Close()
			localConn.Close()

			if err != nil {
				errChan <- err
				return
			}
			if code != Success {
				if code == TimedOut {
					continue
				}
			} else {
				fmt.Printf("Unexcepted error code %d\n", code)
			}
		}
	}()

	return errChan, nil
}

func pass(receiveConn *net.UDPConn, sendConn *net.UDPConn, sendAddr *net.UDPAddr, onReceived func(addr *net.UDPAddr) bool) (Error, error) {
	accepted := false
	bufferChan := make(chan []byte, BUFFER_SIZE)
	errChan := make(chan error)

	ticker := time.NewTicker(1000 * time.Millisecond)

	defer ticker.Stop()
	// 受信
	go func() {
		for {
			buf := make([]byte, BUFFER_SIZE)
			len, addr, err := receiveConn.ReadFromUDP(buf)
			if err != nil {
				errChan <- err
				return
			}
			if !accepted {
				accepted = onReceived(addr)
			}
			bufferChan <- buf[:len]
		}
	}()

	tickCount := 0

	// 送信
	for {
		select {
		case buf := <-bufferChan:
			var err error
			if sendAddr != nil {
				_, err = sendConn.WriteTo(buf, sendAddr)
			} else {
				_, err = sendConn.Write(buf)
			}
			if err != nil {
				return Runtime, err
			}
			tickCount = 0
		case err := <-errChan:
			return Runtime, err
		case <-ticker.C:
			if tickCount > 2 {
				return TimedOut, nil
			}
			tickCount++
		}
	}

}

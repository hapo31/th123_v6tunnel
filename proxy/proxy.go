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
	LocalConn *net.UDPConn

	RemoteAddr *net.UDPAddr
	RemoteConn *net.UDPConn

	receiving bool

	RecvChannel  chan ProxyChannel
	ErrorChannel chan error
	AbortChannel chan bool
}

func New(recvClientPort int) (Proxy, error) {
	p := Proxy{}
	p.LocalAddr = &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: recvClientPort,
	}

	p.ErrorChannel = make(chan error)
	p.receiving = false

	return p, nil
}

func (p *Proxy) StartClient(sendAddrStr string) (chan error, error) {

	errChan := make(chan error)
	// 天則クライアントの待ち受け及びリモートからの通信待ち受け
	go func() {
		for {
			remoteAddr, err := net.ResolveUDPAddr("udp", sendAddrStr)
			if err != nil {
				log.Fatal(err)
				errChan <- err
				return
			}

			localConn, err := net.ListenUDP("udp", p.LocalAddr)
			if err != nil {
				if err != nil {
					log.Fatal(err)
					errChan <- err
					return
				}
			}

			remoteConn, err := net.DialUDP("udp", nil, remoteAddr)
			if err != nil {
				log.Fatal(err)
				errChan <- err
				return
			}
			code, err := pass(localConn, remoteConn, nil, func(r *net.UDPAddr) {
				go pass(remoteConn, localConn, r, func(_ *net.UDPAddr) {})
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

func (p *Proxy) StartServer(proxyPort int) (chan error, error) {
	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("[::]:%d", proxyPort+1))
	fmt.Printf("wait from port %d\n", remoteAddr.Port)
	if err != nil {
		return nil, err
	}

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

			code, err := pass(remoteConn.(*net.UDPConn), localConn.(*net.UDPConn), nil, func(r *net.UDPAddr) {
				go pass(localConn.(*net.UDPConn), remoteConn.(*net.UDPConn), r, func(_ *net.UDPAddr) {})
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

func pass(receiveConn *net.UDPConn, sendConn *net.UDPConn, sendAddr *net.UDPAddr, onReceived func(addr *net.UDPAddr)) (Error, error) {
	accepted := false
	bufferChan := make(chan (struct {
		buf []byte
		len int
	}), BUFFER_SIZE)
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
				onReceived(addr)
				accepted = true
			}
			bufferChan <- struct {
				buf []byte
				len int
			}{buf: buf, len: len}
		}
	}()

	tickCount := 0

	// 送信
	for {
		select {
		case buf := <-bufferChan:
			var err error
			var addr *net.UDPAddr
			if sendAddr != nil {
				addr = sendAddr
				_, err = sendConn.WriteTo(buf.buf[:buf.len], sendAddr)
			} else {
				addr = sendConn.RemoteAddr().(*net.UDPAddr)
				_, err = sendConn.Write(buf.buf[:buf.len])
			}
			fmt.Printf("received: %s\n", receiveConn.LocalAddr().String())
			fmt.Printf("send: %s\n", addr.String())
			if err != nil {
				return Runtime, err
			}
			tickCount = 0
		case err := <-errChan:
			return Runtime, err
		case <-ticker.C:
			if tickCount > 3 {
				return TimedOut, nil
			}
			tickCount++
		}
	}

}

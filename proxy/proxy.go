package proxy

import (
	"context"
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
	Stop
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

func New(th123Addr string) (Proxy, error) {
	p := Proxy{}

	addr, err := net.ResolveUDPAddr("udp", th123Addr)
	if err != nil {
		panic(err.Error())
	}
	p.LocalAddr = addr

	return p, nil
}

func (p *Proxy) StartClient(ctx context.Context, sendAddrStr string) (chan error, error) {
	errChan := make(chan error)
	remoteAddr, err := net.ResolveUDPAddr("udp", sendAddrStr)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	var remoteConn net.Conn
	var localConn net.Conn

	// 天則クライアントの待ち受け及びリモートからの通信待ち受け
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("stop client")
				remoteConn.Close()
				localConn.Close()
				return
			default:
				localConn, err = net.ListenUDP("udp", p.LocalAddr)
				if err != nil {
					log.Fatal(err)
					errChan <- err
					return
				}

				remoteConn, err = net.DialUDP("udp", nil, remoteAddr)
				if err != nil {
					log.Fatal(err)
					errChan <- err
					return
				}

				code, err := pass(ctx, localConn.(*net.UDPConn), remoteConn.(*net.UDPConn), nil, func(r *net.UDPAddr) bool {
					recvConn, _ := net.ListenUDP("udp", remoteConn.LocalAddr().(*net.UDPAddr))
					go pass(ctx, recvConn, localConn.(*net.UDPConn), r, func(rr *net.UDPAddr) bool { return true })
					return true
				})

				if err != nil {
					errChan <- err
					return
				}
				if code != Success {
					if code == TimedOut {
						remoteConn.Close()
						localConn.Close()
						continue
					}
				}
			}
		}
	}()

	return errChan, nil

}

func (p *Proxy) StartServer(ctx context.Context, proxyPort int) (chan error, error) {

	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("[::]:%d", proxyPort))
	if err != nil {
		return nil, err
	}

	var remoteConn net.PacketConn
	var localConn net.Conn

	errChan := make(chan error)
	// リモートからの通信待ち受け
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("stop server")
				remoteConn.Close()
				localConn.Close()
				return
			default:
				remoteConn, err = net.ListenPacket("udp", remoteAddr.String())
				if err != nil {
					log.Fatal(err)
					errChan <- err
					return
				}

				localConn, err = net.Dial("udp", p.LocalAddr.String())
				if err != nil {
					log.Fatal(err)
					errChan <- err
					return
				}

				code, err := pass(ctx, remoteConn.(*net.UDPConn), localConn.(*net.UDPConn), nil, func(r *net.UDPAddr) bool {
					fmt.Printf("accepted:%s\n", r.AddrPort().String())
					go pass(ctx, localConn.(*net.UDPConn), remoteConn.(*net.UDPConn), r, func(_ *net.UDPAddr) bool { return true })
					return true
				})

				if err != nil {
					errChan <- err
					return
				}
				if code != Success {
					if code == TimedOut {
						remoteConn.Close()
						localConn.Close()
						continue
					}
				} else {
					fmt.Printf("Unexcepted error code %d\n", code)
				}
			}
		}
	}()

	return errChan, nil
}

func pass(ctx context.Context, receiveConn *net.UDPConn, sendConn *net.UDPConn, sendAddr *net.UDPAddr, onReceived func(addr *net.UDPAddr) bool) (Error, error) {
	accepted := false
	bufferChan := make(chan []byte, BUFFER_SIZE)
	errChan := make(chan error)

	ticker := time.NewTicker(1000 * time.Millisecond)

	defer ticker.Stop()
	// 受信
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
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
		case <-ctx.Done():
			return Stop, nil
		}
	}
}

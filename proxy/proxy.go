package proxy

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

type Error int

type Mode int

const (
	Client Mode = iota
	Server
)

type State int

const (
	Wait State = iota
	SendingClientPort
	WaitingClientPort
	AcceptedClientPort
	StartConnection
	EndConnection
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

	state State

	receiving bool

	RecvChannel chan ProxyChannel
}

func New(recvClientPort int) (Proxy, error) {
	p := Proxy{}
	p.LocalAddr = &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: recvClientPort,
	}

	p.state = Wait
	p.receiving = false

	return p, nil
}


func (p *Proxy) StartClient(sendAddrStr string) (chan bool, error) {

	// 天則クライアントの待ち受け及びリモートからの通信待ち受け
localConn, err := net.ListenUDP("udp", p.LocalAddr)
	if err != nil {
		return nil, err
	}
	p.LocalConn = localConn

	fmt.Println("start local server")

	sendAddr, err := parseIP(sendAddrStr)
	if err != nil {
		return nil, err
	}
	p.RemoteAddr = sendAddr

	remoteSendConn, err := net.Dial("udp", p.RemoteAddr.String())
	fmt.Printf("Remote send:%v\n", p.RemoteAddr.String())
	if err != nil {
		return nil, err
	}
	fmt.Println("start remote client")

	p.RemoteConn = remoteSendConn.(*net.UDPConn)

	receivedPortInfoChannel := make(chan bool)

	handshake := MakeHandshake(p.LocalAddr)

	p.state = SendingClientPort
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		bytes, err := EncodeToBytes(handshake)
		fmt.Printf("%dbytes \n", len(bytes))
		if err != nil {
			log.Fatal(err)
			return
		}
		for {
			select {
			case <-ticker.C:
				fmt.Println("write")
				p.RemoteConn.Write(bytes)
			case <-receivedPortInfoChannel:
				fmt.Println("end")
				return
			}
		}
	}()

	go func() {
		for {
			defer func() { p.state = AcceptedClientPort }()
			buf := make([]byte, 256)
			// 天則クライアントへ送信すべきポート番号を受信する
			for {
				n, err := p.RemoteConn.Read(buf)
				if err != nil {
					log.Fatal(err)
					return
				}
				fmt.Printf("%d\n", n)
				if n > 0 {
					break
				}
			}

			handshake, err := DecodeFromBytes(buf)
			if err != nil {
				log.Fatal(err)
				return
			}

			p.LocalAddr = HandShakeToUDPAddr(handshake)
			fmt.Println("close chann")
			close(receivedPortInfoChannel)
		}
	}()

	return receivedPortInfoChannel, nil
}

// TODO サーバーモード側の処理を書く


func parseIP(addrStr string) (*net.UDPAddr, error) {
	ip, portStr, err := net.SplitHostPort(addrStr)

	if err != nil {
		return nil, err
	}

	port, err := strconv.ParseInt(portStr, 10, 16)

	if err != nil {
		return nil, err
	}

	return &net.UDPAddr{
		IP: net.ParseIP(ip),
		Port: int(port),
	}, nil

}
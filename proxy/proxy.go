package proxy

import (
	"fmt"
	"net"
	"strconv"
)

type Error int

type Mode int

const BUFFER_SIZE = 128

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

	fmt.Println("start local server")

	remoteAddr, err := parseIP(sendAddrStr)
	if err != nil {
		return nil, err
	}

	remoteConn, err := net.Dial("udp", remoteAddr.String())
	fmt.Printf("Remote send:%v\n", remoteAddr.String())
	if err != nil {
		return nil, err
	}
	fmt.Println("start remote client")

	abortChan, _ := passThroughPacket(remoteConn.(*net.UDPConn), localConn)

	return abortChan, nil

}

func (p *Proxy) StartServer(proxyPort int) (chan bool, error) {
	remoteAddr, err := parseIP(fmt.Sprintf("[::1]:%d", proxyPort))
	if err != nil {
		return nil, err
	}

	remoteConn, err := net.ListenUDP("udp", remoteAddr)
	if err != nil {
		return nil, err
	}

	localConn, err := net.Dial("udp", p.LocalAddr.String())
	if err != nil {
		return nil, err
	}

	abortChan, _ := passThroughPacket(remoteConn, localConn.(*net.UDPConn))

	return abortChan, nil
}

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
		IP:   net.ParseIP(ip),
		Port: int(port),
	}, nil

}

func passThroughPacket(remoteConn *net.UDPConn, localConn *net.UDPConn) (chan bool, chan error) {
	abortChan := make(chan bool)
	errorChan := make(chan error)

	go func() {
		buf := make([]byte, BUFFER_SIZE)
		// リモートからデータ読んでローカルへ送信
		for {
			len, addr, err := remoteConn.ReadFromUDP(buf)
			if err != nil {
				errorChan <- err
				return
			}
			fmt.Printf("->th123 %d\n", len)
			localConn.WriteToUDP(buf, addr)
		}
	}()

	go func() {
		for {
			// ローカルからデータ読んでリモートへ送信
			buf := make([]byte, BUFFER_SIZE)
			len, addr, err := localConn.ReadFromUDP(buf)
			if err != nil {
				errorChan <- err
				return
			}
			fmt.Printf("<-th123 %d\n", len)
			remoteConn.WriteToUDP(buf, addr)
		}
	}()

	return abortChan, errorChan
}

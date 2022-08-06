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

	fmt.Printf("start local server:%s\n", localConn.LocalAddr().String())

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

	abortChan, _ := passThroughPacket(remoteConn.(*net.UDPConn), localConn, nil)

	return abortChan, nil

}

func (p *Proxy) StartServer(proxyPort int) (chan bool, error) {
	remoteAddr, err := parseIP(fmt.Sprintf("[::]:%d", proxyPort+1))
	fmt.Printf("Receive from %d\n", remoteAddr.Port)
	if err != nil {
		return nil, err
	}

	remoteConn, err := net.ListenPacket("udp", remoteAddr.String())
	if err != nil {
		return nil, err
	}

	localConn, err := net.Dial("udp", p.LocalAddr.String())
	if err != nil {
		return nil, err
	}

	abortChan, _ := passThroughPacket(remoteConn.(*net.UDPConn), localConn.(*net.UDPConn), remoteConn.LocalAddr().(*net.UDPAddr))

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

func passThroughPacket(remoteConn *net.UDPConn, localConn *net.UDPConn, recvRemoteAddr *net.UDPAddr) (chan bool, chan error) {
	abortChan := make(chan bool)
	errorChan := make(chan error)

	// recvRemoteAddrChan := make(chan net.UDPAddr)

	acceptedRemoteToLocal := false
	acceptedLocalToRemote := false

	if addr := remoteConn.LocalAddr(); addr != nil {
		fmt.Printf("Remote local addr:%s\n", addr.String())
	}
	if addr := remoteConn.RemoteAddr(); addr != nil {
		fmt.Printf("Remote remote addr:%s\n", addr.String())
	}
	go func() {
		defer remoteConn.Close()
		buf := make([]byte, BUFFER_SIZE)
		// リモートからデータ読んでローカルへ送信
		for {
			len, addr, err := remoteConn.ReadFromUDP(buf)
			if err != nil {
				errorChan <- err
				return
			}
			if !acceptedRemoteToLocal {
				fmt.Printf("receive from remote[%s]\n", addr.String())
				acceptedRemoteToLocal = true
			}
			// fmt.Printf("->th123 %d\n", len)
			localConn.Write(buf[:len])
		}
	}()

	go func() {
		defer localConn.Close()
		for {
			// ローカルからデータ読んでリモートへ送信
			buf := make([]byte, BUFFER_SIZE)
			len, addr, err := localConn.ReadFromUDP(buf)
			if err != nil {
				errorChan <- err
				return
			}
			if !acceptedLocalToRemote {
				fmt.Printf("receive from local[%s]\n", addr.String())
				acceptedLocalToRemote = true
			}
			// fmt.Printf("<-th123 %d\n", len)
			remoteConn.Write(buf[:len])
		}
	}()

	return abortChan, errorChan
}

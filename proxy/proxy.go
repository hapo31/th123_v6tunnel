package proxy

import (
	"fmt"
	"log"
	"net"
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

func (p *Proxy) StartClient(sendAddrStr string) (chan bool, chan error, error) {
	// 天則クライアントの待ち受け及びリモートからの通信待ち受け
	localConn, err := net.ListenUDP("udp", p.LocalAddr)
	if err != nil {
		log.Fatal(err)
		return nil, nil, err
	}

	fmt.Printf("start local server:%s\n", localConn.LocalAddr().String())

	remoteAddr, err := net.ResolveUDPAddr("udp", sendAddrStr)
	if err != nil {
		log.Fatal(err)
		return nil, nil, err
	}

	remoteConn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Fatal(err)
		return nil, nil, err
	}

	fmt.Printf("Remote send:%v\n", remoteAddr.String())

	fmt.Println("start remote client")

	abortChan, errChan, err := passThroughPacket(remoteConn, localConn, remoteConn.LocalAddr().(*net.UDPAddr), true)
	if err != nil {
		log.Fatal(err)
		return nil, nil, err
	}

	return abortChan, errChan, nil

}

func (p *Proxy) StartServer(proxyPort int) (chan bool, chan error, error) {
	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("[::]:%d", proxyPort+1))
	fmt.Printf("Receive from %d\n", remoteAddr.Port)
	if err != nil {
		return nil, nil, err
	}

	remoteConn, err := net.ListenPacket("udp", remoteAddr.String())
	if err != nil {
		log.Fatal(err)
		return nil, nil, err
	}

	localConn, err := net.Dial("udp", p.LocalAddr.String())
	if err != nil {
		log.Fatal(err)
		return nil, nil, err
	}

	abortChan, errChan, err := passThroughPacket(remoteConn.(*net.UDPConn), localConn.(*net.UDPConn), nil, false)
	if err != nil {
		log.Fatal(err)
		return nil, nil, err
	}

	return abortChan, errChan, nil
}

func passThroughPacket(remoteConn *net.UDPConn, localConn *net.UDPConn, recvRemoteAddr *net.UDPAddr, knownRemoteAddr bool) (chan bool, chan error, error) {
	abortChan := make(chan bool)
	errorChan := make(chan error)

	var receiveRemoteConn *net.UDPConn
	if recvRemoteAddr == nil {
		receiveRemoteConn = remoteConn
	} else {
		conn, err := net.ListenUDP("udp", recvRemoteAddr)
		if err != nil {
			log.Fatal(err)
			return nil, nil, err
		}
		receiveRemoteConn = conn
	}

	var sendRemoteConn *net.UDPConn
	if knownRemoteAddr {
		sendRemoteConn = remoteConn
	} else {
		conn, err := net.DialUDP("udp", remoteConn.LocalAddr().(*net.UDPAddr), remoteConn.RemoteAddr().(*net.UDPAddr))
		if err != nil {
			log.Fatal(err)
			return nil, nil, err
		}
		sendRemoteConn = conn
	}

	acceptedRemoteToLocal := false
	acceptedLocalToRemote := false

	if addr := receiveRemoteConn.LocalAddr(); addr != nil {
		fmt.Printf("Remote local addr:%s\n", addr.String())
	}
	if addr := sendRemoteConn.RemoteAddr(); addr != nil {
		fmt.Printf("Remote remote addr:%s\n", addr.String())
	}
	go func() {
		defer receiveRemoteConn.Close()
		// リモートからデータ読んでローカルへ送信
		for {
			buf := make([]byte, BUFFER_SIZE)
			len, addr, err := receiveRemoteConn.ReadFrom(buf)
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
		defer sendRemoteConn.Close()
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
			sendRemoteConn.Write(buf[:len])
		}
	}()

	return abortChan, errorChan, nil
}

package proxy

import (
	"log"
	"net"
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
	LocalSendAddr *net.UDPAddr
	LocalSendConn *net.UDPConn
	LocalRecvAddr *net.UDPAddr
	LocalRecvConn *net.UDPConn

	RemoteSendAddr *net.UDPAddr
	RemoteSendConn *net.UDPConn
	RemoteRecvAddr *net.UDPAddr
	RemoteRecvConn *net.UDPConn

	state State

	receiving bool

	RecvChannel chan ProxyChannel
}

func (p *Proxy) MakeProxy(recvPort int, remoteReceiveAddr *net.UDPAddr) (int, error) {
	// 天則クライアントの待ち受け及びリモートからの通信待ち受け
	p.LocalRecvAddr = &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: recvPort,
	}

	localRecvConn, err := net.ListenUDP("udp", p.LocalRecvAddr)
	if err != nil {
		log.Fatal(err)
		return 1, err
	}
	p.LocalRecvConn = localRecvConn
	p.RemoteRecvAddr = remoteReceiveAddr
	remoteRecvConn, err := net.ListenUDP("upd6", p.RemoteRecvAddr)
	if err != nil {
		log.Fatal(err)
		return 1, err
	}

	p.RemoteRecvConn = remoteRecvConn

	p.state = Wait
	p.receiving = false

	return 0, nil
}

func (p *Proxy) StartClient(sendAddr *net.UDPAddr) {
	if sendAddr == nil {
		log.Fatal("Must be set remote addr")
		return
	}

	p.RemoteRecvAddr = &net.UDPAddr{
		IP:   net.ParseIP("::1"),
		Port: sendAddr.Port + 1,
	}

	remoteRecvConn, err := net.ListenUDP("udp", p.RemoteRecvAddr)
	if err != nil {
		log.Fatal(err)
		return
	}
	p.RemoteRecvConn = remoteRecvConn

	p.RemoteSendAddr = sendAddr

	remoteSendConn, err := net.Dial("udp", p.RemoteSendAddr.String())
	if err != nil {
		log.Fatal(err)
		return
	}
	p.RemoteSendConn = remoteSendConn.(*net.UDPConn)

	receivedPortInfoChannel := make(chan bool)

	handshake := MakeHandshake(p.LocalRecvAddr)

	go func() {
		p.state = SendingClientPort
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		bytes, err := EncodeToBytes(handshake)
		if err != nil {
			log.Fatal(err)
			return
		}
		for {
			select {
			case <-receivedPortInfoChannel:
				return
			case <-ticker.C:
				p.RemoteSendConn.Write(bytes)
			}
		}
	}()

	go func() {
		for {
			defer func() { p.state = AcceptedClientPort }()
			buf := make([]byte, 64)
			_, _, err := p.RemoteRecvConn.ReadFromUDP(buf)
			if err != nil {
				log.Fatal(err)
				return
			}
			// 天則クライアントへ送信すべきポート番号を受信する
			handshake, err := DecodeFromBytes(buf)
			if err != nil {
				log.Fatal(err)
				return
			}

			p.LocalSendAddr = HandShakeToUDPAddr(handshake)
			close(receivedPortInfoChannel)
		}
	}()
}

// TODO サーバーモード側の処理を書く

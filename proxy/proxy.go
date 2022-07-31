package proxy

import (
	"errors"
	"fmt"
	"log"
	"net"
)

type Error int

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
	Buf []byte
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
		IP: net.ParseIP("127.0.0.1"),
		Port: recvPort,
	}

	localRecvConn, err := net.ListenUDP("udp", p.LocalRecvAddr)
	if err != nil {
		log.Fatal(err)
		return 1, err
	}
	p.LocalRecvConn = localRecvConn
	p.RemoteRecvAddr = remoteReceiveAddr
	remoteRecvConn,err := net.ListenUDP("upd6", p.RemoteRecvAddr)
	if err != nil {
		log.Fatal(err)
		return 1, err
	}

	p.RemoteRecvConn = remoteRecvConn

	p.state = Wait
	p.receiving = false

	return 0, nil
}

func (p* Proxy) StartSession() {

	stateChanged := false

	go func() {
		for {
			
		}
	}()

	go func ()  {
		
	}()

}

func (h *Proxy) StartLocalServer(recvAddr *net.UDPAddr) (int, error) {
	// 天則(クライアント)からのパケットが待ち受けられるようにする
	// リモートの th123_tunnel にデータをリレーする
	if h.LocalRecvAddr == nil {
		return 1, errors.New("recv_addr was not set")
	}

	if h.receiving {
		return 1, errors.New("already receiving")
	}

	h.RecvChannel = make(chan ProxyChannel)

	h.LocalRecvAddr = recvAddr
	recvConn, err := net.ListenUDP("udp", h.LocalRecvAddr)
	h.LocalRecvConn = recvConn
	if err != nil {
		return 1, err
	}

	h.RemoteSendAddr = &net.UDPAddr{
		IP: net.ParseIP("127.0.0.1"),
		Port: recvAddr.Port + 1, // th123_tunnel 同士が通信するポートは天則のポート + 1 とする
	}

	remoteSendConn, err := net.Dial("udp", h.RemoteSendAddr.String())
	if err != nil {
		return 1, err
	}
	h.RemoteSendConn = remoteSendConn.(*net.UDPConn)
	
	h.receiving = true
	for {
		if !h.receiving {
			break
		}
		buf := make([]byte, 64)
		n, addr, err := h.LocalRecvConn.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
			return 1, err
		}
		h.RecvChannel <- ProxyChannel { Addr: addr, Buf: buf }
		if h.LocalSendAddr == nil && h.LocalSendConn == nil {
			if addr.Port == h.LocalRecvAddr.Port {
				continue
			}
			fmt.Printf("relay port detected %d\n",addr.Port)
			h.LocalSendAddr = &net.UDPAddr{
				IP: addr.IP,
				Port: addr.Port,
			}
			conn, err := net.Dial("udp", h.LocalSendAddr.String())
			if err != nil {
				log.Fatal(err)
				return 1, err
			}
			h.LocalSendConn = conn.(*net.UDPConn)
		}
		if h.LocalSendConn != nil {
			h.LocalSendConn.Write(buf[:n])
		}
	}

	return 0, nil
}

func (h* Proxy) StopReceive() {
	if h.LocalSendConn != nil {
		h.LocalSendConn.Close()
		h.LocalSendConn = nil
	}
	if h.LocalRecvConn != nil {
		h.LocalRecvConn.Close()
		h.LocalRecvConn = nil
	}

	h.receiving = false

}
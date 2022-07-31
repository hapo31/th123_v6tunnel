package main

import (
	"errors"
	"fmt"
	"net"
)

type Proxy struct {
	send_addr *net.UDPAddr
	send_conn *net.UDPConn
	recv_addr *net.UDPAddr
	recv_conn *net.UDPConn

	receiving bool

}

func (h *Proxy) startReceive() (int, error) {
	if h.recv_addr == nil {
		return 1, errors.New("recv_addr was not set")
	}

	if h.receiving {
		return 1, errors.New("already receiving")
	}

	recv_conn, err := net.ListenUDP("udp", h.recv_addr)
	h.recv_conn = recv_conn
	if err != nil {
		return 1, err
	}

	go func() {
		if !h.receiving {
			return
		}
		buf := make([]byte, 64)
		for {
			n, _, err := h.recv_conn.ReadFromUDP(buf)
			if err != nil {
				fmt.Printf("ReadFromUDP failed. %v", err)
				return
			}
			if h.send_conn != nil {
				h.send_conn.Write(buf[:n])
			}
		}
	}()

	return 0, nil
}

func (h* Proxy) stopReceive() {

}
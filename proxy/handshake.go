package proxy

import (
	"bytes"
	"encoding/gob"
	"errors"
	"net"
)

func getHeaderBytes() [2]byte {
	header := [2]byte{0x08, 0x31}
	return header
}

type ProxyHandshake struct {
	Header [2]byte
	IP     [16]byte
	Port   int
}

func MakeHandshake(addr *net.UDPAddr) ProxyHandshake {
	handshake := ProxyHandshake{
		Header: getHeaderBytes(),
	}
	copy(handshake.IP[:], addr.IP)
	handshake.Port = addr.Port
	return handshake
}

func DecodeFromBytes(src []byte) (*ProxyHandshake, error) {
	var buf bytes.Buffer
	buf.Write(src)

	var dest ProxyHandshake
	decoder := gob.NewDecoder(&buf)
	err := decoder.Decode(&dest)

	if err != nil {
		return nil, err
	}

	if dest.Header[0] != 0x08 || dest.Header[1] != 0x31 {
		return nil, errors.New("Header not match")
	}

	return &dest, nil
}

func EncodeToBytes(src ProxyHandshake) ([]byte, error) {
	var buf bytes.Buffer

	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(src)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func HandShakeToUDPAddr(src *ProxyHandshake) *net.UDPAddr {
	return &net.UDPAddr{
		IP:   src.IP[:],
		Port: int(src.Port),
	}
}

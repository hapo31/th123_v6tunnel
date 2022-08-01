package proxy

import (
	"fmt"
	"net"
	"testing"

	"happo31.jp/th123tunnel/proxy"
)


func TestHandShakeEncodeDecode (t *testing.T) {
	
	t.Run("", func(t *testing.T) {
		addr := &net.UDPAddr{
			IP: net.ParseIP("255.255.255.255"),
			Port: 10800,
		}
	
		p := proxy.MakeHandshake(addr)
	
		bytes, err := proxy.EncodeToBytes(p)
		if err != nil {
			t.Errorf("EncodeToBytes(p) err is not nil")
			t.Fail();
		}
	
		for _, v := range bytes {
			fmt.Printf("%02x,", v)
		}
		fmt.Println()
		
	
		pp, err := proxy.DecodeFromBytes(bytes)
	
		if err != nil {
			t.Errorf("DecodeFromBytes(bytes) err is not nil")
			t.Fail();
		}
	
		destAddr := net.UDPAddr {
			IP: pp.IP[:],
			Port: int(pp.Port),
		}

		if destAddr.String() != "255.255.255.255:10800" {
			t.Fail();
		}
	})
}
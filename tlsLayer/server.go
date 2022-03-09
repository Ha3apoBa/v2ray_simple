package tlsLayer

import (
	"crypto/tls"
	"net"

	"github.com/hahahrfool/v2ray_simple/common"
)

type Server struct {
	addr      string
	tlsConfig *tls.Config
}

func NewServer(hostAndPort, host, certFile, keyFile string, isInsecure bool) (*Server, error) {

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	s := &Server{
		addr: hostAndPort,
		tlsConfig: &tls.Config{
			InsecureSkipVerify: isInsecure,
			ServerName:         host,
			Certificates:       []tls.Certificate{cert},
		},
	}

	return s, nil
}

func (s *Server) Handshake(underlay net.Conn) (tlsConn *tls.Conn, err error) {
	tlsConn = tls.Server(underlay, s.tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		return tlsConn, common.NewErr("tls握手失败", err)
	}

	return

}

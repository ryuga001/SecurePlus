package receiver

import (
	"context"
	"net"

	"github.com/emersion/go-smtp"
	"golang.org/x/net/netutil"

	"dpdp-backend/internal/config"
)

type Server struct {
	smtp           *smtp.Server
	addr           string
	maxConnections int
}

func NewServer(cfg config.SMTPServer, backend *Backend, heloHost string) *Server {
	server := smtp.NewServer(backend)

	server.Addr = cfg.Addr
	server.Domain = heloHost
	server.ReadTimeout = cfg.ReadTimeout
	server.WriteTimeout = cfg.WriteTimeout
	server.MaxMessageBytes = cfg.MaxSize
	server.MaxRecipients = cfg.MaxRecipients
	server.MaxLineLength = 4096
	server.AllowInsecureAuth = true

	return &Server{smtp: server, addr: cfg.Addr, maxConnections: cfg.MaxConnections}
}

func (s *Server) Addr() string { return s.addr }

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	if s.maxConnections > 0 {
		listener = netutil.LimitListener(listener, s.maxConnections)
	}

	return s.smtp.Serve(listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.smtp.Shutdown(ctx)
}

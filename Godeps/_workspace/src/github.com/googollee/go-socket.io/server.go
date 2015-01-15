package socketio

import (
	"github.com/rckclmbr/goportify/Godeps/_workspace/src/github.com/googollee/go-engine.io"
	"net/http"
	"time"
)

// Server is the server of socket.io.
type Server struct {
	*namespace
	broadcast BroadcastAdaptor
	eio       *engineio.Server
}

// NewServer returns the server supported given transports. If transports is nil, server will use ["polling", "websocket"] as default.
func NewServer(transportNames []string) (*Server, error) {
	eio, err := engineio.NewServer(transportNames)
	if err != nil {
		return nil, err
	}
	ret := &Server{
		namespace: newNamespace(newBroadcastDefault()),
		eio:       eio,
	}
	go ret.loop()
	return ret, nil
}

// SetPingTimeout sets the timeout of ping. When time out, server will close connection. Default is 60s.
func (s *Server) SetPingTimeout(t time.Duration) {
	s.eio.SetPingTimeout(t)
}

// SetPingInterval sets the interval of ping. Default is 25s.
func (s *Server) SetPingInterval(t time.Duration) {
	s.eio.SetPingInterval(t)
}

// SetAllowRequest sets the middleware function when establish connection. If it return non-nil, connection won't be established. Default will allow all request.
func (s *Server) SetAllowRequest(f func(*http.Request) error) {
	s.eio.SetAllowRequest(f)
}

// SetAllowUpgrades sets whether server allows transport upgrade. Default is true.
func (s *Server) SetAllowUpgrades(allow bool) {
	s.eio.SetAllowUpgrades(allow)
}

// SetCookie sets the name of cookie which used by engine.io. Default is "io".
func (s *Server) SetCookie(prefix string) {
	s.eio.SetCookie(prefix)
}

// SetAdaptor sets the adaptor of broadcast. Default is in-process broadcast implement.
func (s *Server) SetAdaptor(adaptor BroadcastAdaptor) {
	s.namespace = newNamespace(adaptor)
}

// ServeHTTP handles http request.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.eio.ServeHTTP(w, r)
}

// Server level broadcasts function.
func (s *Server) BroadcastTo(room, message string, args ...interface{}) {
	s.namespace.BroadcastTo(room, message, args...)
}

func (s *Server) loop() {
	for {
		conn, err := s.eio.Accept()
		if err != nil {
			return
		}
		s := newSocket(conn, s.baseHandler)
		go func(s *socket) {
			s.loop()
		}(s)
	}
}

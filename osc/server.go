package osc

import (
	"fmt"
	"log"

	"github.com/hypebeast/go-osc/osc"
)

type Server struct {
	srv *osc.Server
	out chan string
}

func NewServer(port int) *Server {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	d := osc.NewStandardDispatcher()
	s := &Server{
		out: make(chan string, 100),
		srv: &osc.Server{
			Addr:       addr,
			Dispatcher: d,
		},
	}
	d.AddMsgHandler("/play", s.HandlePlay)
	return s

}

func (s *Server) Start() error {
	log.Printf("Starting OSC server on %s\n", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func (s *Server) HandlePlay(msg *osc.Message) {
	select {
	case s.out <- msg.String():
	default:
		log.Println("dropped osc message")
	}
}
func (s *Server) Out() chan string {
	return s.out
}

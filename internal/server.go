package internal

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Clients    map[net.Conn]*Client
	Mtx        sync.RWMutex
	Broadcast  chan string
	Register   chan *Client
	Unregister chan net.Conn
}

func NewServer() *Server {
	return &Server{
		Clients:    make(map[net.Conn]*Client),
		Broadcast:  make(chan string, 256),
		Register:   make(chan *Client),
		Unregister: make(chan net.Conn),
	}
}

func (s *Server) Run() {
	for {
		select {
		case client := <-s.Register:
			s.Mtx.Lock()
			s.Clients[client.Conn] = client
			s.Mtx.Unlock()
			go client.WritePump()

		case conn := <-s.Unregister:
			s.Mtx.Lock()
			if client, ok := s.Clients[conn]; ok {
				delete(s.Clients, conn)
				close(client.Send)
			}
			s.Mtx.Unlock()

		case msg := <-s.Broadcast:
			s.Mtx.RLock()
			for _, client := range s.Clients {
				select {
				case client.Send <- msg:
				default:
				}
			}
			s.Mtx.RUnlock()
		}
	}
}

func (s *Server) GetClientList() string {
	s.Mtx.RLock()
	defer s.Mtx.RUnlock()
	names := make([]string, 0, len(s.Clients))
	for _, client := range s.Clients {
		if client.Name != "" {
			names = append(names, client.Name)
		}
	}

	return strings.Join(names, ", ")
}

func (s *Server) IsNameTaken(name string) bool {
	s.Mtx.RLock()
	defer s.Mtx.RUnlock()
	for _, client := range s.Clients {
		if client.Name == name {
			return true
		}
	}
	return false
}

func (s *Server) SendPrivateMessage(senderName, recipientName, message string) {
	s.Mtx.RLock()
	defer s.Mtx.RUnlock()

	var sender *Client
	var recipient *Client

	for _, client := range s.Clients {
		if client.Name == senderName {
			sender = client
		}
		if client.Name == recipientName {
			recipient = client
		}
	}

	if recipient == nil {
		if sender != nil {
			sender.Send <- fmt.Sprintf("Пользователь %s не найден", recipientName)
		}
		return
	}

	timestamp := time.Now().Format("15:04:05")

	recipient.Send <- fmt.Sprintf("[%s] Личное от %s: %s", timestamp, senderName, message)

	if sender != nil {
		sender.Send <- fmt.Sprintf("[%s] Личное для %s: %s", timestamp, recipientName, message)
	}
}

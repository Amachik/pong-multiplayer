package network

import (
	"fmt"
	"net"
	"sync"
)

type Server struct {
	Address            string
	ExpectedInviteCode string
	Clients            map[string]*net.UDPAddr
	Lock               sync.Mutex
	// InputUpdate is called when the server receives an input_update message.
	InputUpdate func(msg Message)
	conn        *net.UDPConn
}

func NewServer(address, inviteCode string) *Server {
	return &Server{
		Address:            address,
		ExpectedInviteCode: inviteCode,
		Clients:            make(map[string]*net.UDPAddr),
	}
}

func (s *Server) Start() error {
	udpAddr, err := net.ResolveUDPAddr("udp", s.Address)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	s.conn = conn
	fmt.Println("Server listening on", s.Address)
	buf := make([]byte, 1024)
	for {
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading UDP:", err)
			continue
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		msg, err := DecodeMessage(data)
		if err != nil {
			fmt.Println("Error decoding message:", err)
			continue
		}
		switch msg.Type {
		case MessageTypeHandshake:
			// Validate the invite code.
			if string(msg.Data) != s.ExpectedInviteCode {
				errMsg := Message{Type: MessageTypeError, Data: []byte("Invalid invite code")}
				encoded, _ := EncodeMessage(errMsg)
				s.conn.WriteToUDP(encoded, addr)
				continue
			}
			// Send back handshake success.
			successMsg := Message{Type: MessageTypeHandshakeSuccess, Data: []byte("OK")}
			encoded, _ := EncodeMessage(successMsg)
			s.conn.WriteToUDP(encoded, addr)
			// Add client address.
			s.Lock.Lock()
			s.Clients[addr.String()] = addr
			s.Lock.Unlock()
		case MessageTypeInputUpdate:
			if s.InputUpdate != nil {
				s.InputUpdate(msg)
			}
		case MessageTypePing:
			// Immediately respond with a Pong echoing the ping payload.
			pong := Message{
				Type: MessageTypePong,
				Seq:  msg.Seq,
				Data: msg.Data, // echoing the timestamp that the client sent
			}
			encoded, _ := EncodeMessage(pong)
			s.conn.WriteToUDP(encoded, addr)
		default:
			// Other messages can be handled as needed.
		}
	}
	// (Unreachable)
	// return nil
}

func (s *Server) Broadcast(msg Message) {
	data, err := EncodeMessage(msg)
	if err != nil {
		fmt.Println("Error encoding message:", err)
		return
	}
	s.Lock.Lock()
	defer s.Lock.Unlock()
	for _, addr := range s.Clients {
		s.conn.WriteToUDP(data, addr)
	}
}

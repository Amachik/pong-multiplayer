package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"pong-multiplayer/game"
)

var MeasuredRTT int64

func setMeasuredRTT(rtt int64) {
	atomic.StoreInt64(&MeasuredRTT, rtt)
}

type StateUpdateCallback func(state game.State)

type Client struct {
	Address       string
	RemoteAddr    *net.UDPAddr
	Conn          *net.UDPConn
	OnStateUpdate StateUpdateCallback
}

func NewClient(address string) *Client {
	return &Client{
		Address: address,
	}
}

func (c *Client) Connect(inviteCode string) error {
	// Resolve the server address.
	serverAddr, err := net.ResolveUDPAddr("udp", c.Address)
	if err != nil {
		return err
	}
	// Create a UDP connection (letting the system choose a local port).
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return err
	}
	c.Conn = conn
	c.RemoteAddr = serverAddr

	// Send handshake message.
	handshake := Message{
		Type: MessageTypeHandshake,
		Data: []byte(inviteCode),
	}
	encoded, err := EncodeMessage(handshake)
	if err != nil {
		return err
	}
	if _, err = c.Conn.Write(encoded); err != nil {
		return err
	}
	// Set a read deadline for the handshake response.
	c.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 1024)
	n, _, err := c.Conn.ReadFromUDP(buf)
	if err != nil {
		return fmt.Errorf("failed to receive handshake response: %v", err)
	}
	data := buf[:n]
	msg, err := DecodeMessage(data)
	if err != nil {
		return fmt.Errorf("failed to decode handshake response: %v", err)
	}
	if msg.Type == MessageTypeError {
		return fmt.Errorf("handshake error: %s", string(msg.Data))
	}
	// Reset deadline.
	c.Conn.SetReadDeadline(time.Time{})
	go c.listen()

	// Start sending periodic pings
	go func() {
		var pingSeq uint32 = 0
		for {
			pingSeq++
			ts := time.Now().UnixNano()
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.BigEndian, ts)
			pingMsg := Message{
				Type: MessageTypePing,
				Seq:  pingSeq,
				Data: buf.Bytes(),
			}
			encoded, err := EncodeMessage(pingMsg)
			if err != nil {
				fmt.Println("Error encoding ping:", err)
			} else {
				_, err = c.Conn.Write(encoded)
				if err != nil {
					fmt.Println("Error sending ping:", err)
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()

	return nil
}

func (c *Client) listen() {
	var lastSeq uint32 = 0
	buf := make([]byte, 1024)
	for {
		n, _, err := c.Conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		msg, err := DecodeMessage(data)
		if err != nil {
			fmt.Println("Error decoding message:", err)
			continue
		}
		if msg.Type == MessageTypeStateUpdate {
			// Discard if packet is older than the last processed one.
			if msg.Seq <= lastSeq {
				continue
			}
			lastSeq = msg.Seq

			var state game.State
			reader := bytes.NewReader(msg.Data)
			var ballX, ballY, ballVX, ballVY, p1X, p1Y, p2X, p2Y float32
			var scoreLeft, scoreRight int32
			if err := binary.Read(reader, binary.BigEndian, &ballX); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			if err := binary.Read(reader, binary.BigEndian, &ballY); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			if err := binary.Read(reader, binary.BigEndian, &ballVX); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			if err := binary.Read(reader, binary.BigEndian, &ballVY); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			if err := binary.Read(reader, binary.BigEndian, &p1X); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			if err := binary.Read(reader, binary.BigEndian, &p1Y); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			if err := binary.Read(reader, binary.BigEndian, &p2X); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			if err := binary.Read(reader, binary.BigEndian, &p2Y); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			if err := binary.Read(reader, binary.BigEndian, &scoreLeft); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			if err := binary.Read(reader, binary.BigEndian, &scoreRight); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			// Read the timestamp (int64)
			var timestamp int64
			if err := binary.Read(reader, binary.BigEndian, &timestamp); err != nil {
				fmt.Println("binary read error:", err)
				continue
			}
			state = game.State{
				BallX:      ballX,
				BallY:      ballY,
				BallVX:     ballVX,
				BallVY:     ballVY,
				P1X:        p1X,
				P1Y:        p1Y,
				P2X:        p2X,
				P2Y:        p2Y,
				ScoreLeft:  int(scoreLeft),
				ScoreRight: int(scoreRight),
				Timestamp:  timestamp,
			}
			if c.OnStateUpdate != nil {
				c.OnStateUpdate(state)
			}
		} else if msg.Type == MessageTypePong {
			var sentTime int64
			reader := bytes.NewReader(msg.Data)
			if err := binary.Read(reader, binary.BigEndian, &sentTime); err == nil {
				rtt := time.Now().UnixNano() - sentTime
				// Update a global RTT variable (using atomic operations in production)
				setMeasuredRTT(rtt) // implement this function as needed
			}
			continue
		} else {
			fmt.Printf("Received unknown message type: %d\n", msg.Type)
		}
	}
}

func (c *Client) Send(msg Message) error {
	data, err := EncodeMessage(msg)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(data)
	return err
}

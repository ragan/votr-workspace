package votr

import (
	"github.com/gorilla/websocket"
	"log"
	"time"
)

type User struct {
	// client connection
	conn *websocket.Conn
	// messages from room broadcast are written here
	msg  chan []byte
	vote int
}

func NewUser(c *websocket.Conn) *User {
	return &User{
		conn: c,
		msg:  make(chan []byte),
	}
}

// Routine writes messages to client.
func (u *User) write(conn *websocket.Conn) {
	ticker := time.NewTicker(tenSec)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		// Incoming messages are being sent to recipient
		case msg := <-u.msg:
			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("NextWriter error: %v", err)
				return
			}
			w.Write(msg)
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			log.Printf("Ping to %s", conn.RemoteAddr())
			conn.SetWriteDeadline(time.Now().Add(tenSec))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (u *User) read(broadcast chan []byte, delete chan *User) {
	defer func() {
		u.conn.Close()
	}()
	u.conn.SetReadLimit(512)
	u.conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	u.conn.SetPongHandler(func(string) error {
		log.Printf("Pong from %s", u.conn.RemoteAddr())
		u.conn.SetReadDeadline(time.Now().Add(20 * time.Second))
		return nil
	})
	for {
		// message incoming from client
		_, msg, err := u.conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			delete <- u
			return
		}
		broadcast <- msg
	}
}
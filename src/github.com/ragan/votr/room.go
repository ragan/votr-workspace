package votr

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Every message must contain its type
type MessageType int

const (
	// Message occurs when user votes
	VoteMsg MessageType = iota
	// Message occurs when user enters or leaves room
	StatusMsg
)

// Represents messages sent between users.
type Message struct {
	t MessageType
}

var rooms = make(map[string]*Room)

type Room struct {
	users          map[*User]bool
	unregisterChan chan *User
	broadcastChan  chan []byte
}

// Structure representing new incoming room connection
type roomConn struct {
	ws     *websocket.Conn
	roomId string
}

func init() {
	// Debug routine
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				var count = 0
				for _, room := range rooms {
					count += len(room.users)
				}
				log.Printf("Users count is: \"%v\"", count)
				log.Printf("Rooms count is: \"%v\"", len(rooms))
			}
		}
	}()
}

func NewRoom() string {
	id := RoomId(GetIdLen())
	log.Printf("Creating new room with id: %s", id)
	room := &Room{
		make(map[*User]bool),
		make(chan *User),
		make(chan []byte),
	}
	rooms[id] = room
	go room.broadcast()
	return id
}

func GetIdLen() int {
	return 6
}

type RoomInfo struct {
	unregister chan *User
	broadcast  chan []byte
}

// Adding user to room. Room should be created before adding user.
func addUser(u *User, roomId string) (error, RoomInfo) {
	log.Printf("Adding user to room \"%s\"", roomId)
	room, ok := rooms[roomId]
	if !ok {
		return fmt.Errorf("room \"%s\" does not exist", roomId), RoomInfo{}
	}
	room.users[u] = true
	log.Printf("Room users count: \"%v\"", len(room.users))
	return nil, RoomInfo{
		unregister: room.unregisterChan,
		broadcast:  room.broadcastChan,
	}
}

func (r *Room) broadcast() {
	for {
		select {
		case msg := <-r.broadcastChan:
			// New message
			for u := range r.users {
				u.msg <- msg
			}
		case u := <-r.unregisterChan:
			delete(r.users, u)
			u.conn.Close()
		}
	}
}

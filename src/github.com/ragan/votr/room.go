package votr

import (
	"fmt"
	"log"
	"time"
)

var rooms = make(map[string]*Room)

type Room struct {
	users       map[*User]bool
	unregister  chan *User
	newMessages chan []byte
}

func init() {
	// Debug routine
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <- ticker.C:
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
	go room.bCast()
	return id
}

func Exists(roomId string) bool {
	_, ok := rooms[roomId]
	return ok
}

func GetIdLen() int {
	return 6
}

type RoomInfo struct {
	unregister  chan *User
	newMessages chan []byte
}

func AddUser(u *User, roomId string) (error, RoomInfo) {
	log.Printf("Adding user to room \"%s\"", roomId)
	room, ok := rooms[roomId]
	if !ok {
		return fmt.Errorf("room \"%s\" does not exist", roomId), RoomInfo{}
	}
	room.users[u] = true
	room.newMessages <- []byte("New user entered room...")
	log.Printf("Room users count: \"%v\"", len(room.users))
	return nil, RoomInfo{room.unregister, room.newMessages}
}

func (r *Room) bCast() {
	for {
		select {
		case msg := <-r.newMessages:
			// New message
			for u := range r.users {
				u.readChan <- msg
			}
		case u := <-r.unregister:
			log.Printf("Removing user from room...")
			delete(r.users, u)
			r.newMessages <- []byte("User left the room...")
		}
	}
}

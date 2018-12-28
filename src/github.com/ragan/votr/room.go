package votr

import (
	"fmt"
	"log"
)

var rooms = make(map[string]*Room)

type Room struct {
	users       map[*User]bool
	unregister  chan *User
	newMessages chan []byte
}

func NewRoom() string {
	id := RoomId(GetIdLen())
	log.Printf("Creating new room with id: %s", id)
	rooms[id] = &Room{
		make(map[*User]bool),
		make(chan *User),
		make(chan []byte),
	}
	return id
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
	return nil, RoomInfo{room.unregister, room.newMessages}
}

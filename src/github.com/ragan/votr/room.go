package votr

import (
	"fmt"
	"log"
	"time"

	"errors"
	"github.com/gorilla/websocket"
)

// Every message must contain its type
type MessageType int

const (
	// Message occurs when user votes
	VoteMsg MessageType = iota
	// Message occurs when user enters or leaves room
	StatusMsg
	// When message should be ignored by application
	IgnoreMsg
)

// Represents messages sent between users.
type Message struct {
	user  *User
	T     MessageType `json:"type"`
	Value string      `json:"value"`
	UserCount int  `json:"userCount"`
	VoteCount int  `json:"voteCount"`
}

var rooms = make(map[string]*Room)

type Room struct {
	users          map[*User]bool
	unregisterChan chan *User
	broadcastChan  chan Message
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
		make(chan Message),
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
	broadcast  chan Message
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
		// New message
		case msg := <-r.broadcastChan:
			err, m := processMsg(msg)
			if err != nil {
				log.Printf("Processing message error: %s", err)
			} else {
				m.UserCount = len(r.users)
				m.VoteCount = r.countVotes()
				for u := range r.users {
					u.msg <- m
				}
			}
		case u := <-r.unregisterChan:
			delete(r.users, u)
			u.conn.Close()
		}
	}
}

func (r *Room) countVotes() int {
	c := 0
	for u := range r.users {
		if u.vote != FirstVote {
			c++
		}
	}
	return c
}

// Values allowed when placing a vote.
var votes = map[string]int{
	"0": 0,
	"1": 1,
	"2": 2,
	"3": 3,
	"5": 5,
	"8": 8,
}

const (
	UserPlacedVote  = "User placed his vote."
	UserChangedVote = "User changed his vote."
	UserEnteredMsg  = "New user entered room."
	UserLeftMsg     = "User left."
)

// Initial vote value. Indicates a user did not place any vote.
const FirstVote = -1

func processMsg(m Message) (error, Message) {
	switch m.T {
	case VoteMsg:
		// Restrict voting to declared values
		if v, ok := votes[m.Value]; ok {
			if m.user.vote == v {
				// Vote did not change. Ignoring.
				return errors.New("user did not change vote"), Message{}
			}
			s := UserPlacedVote
			// When this is not the first vote
			if m.user.vote != FirstVote {
				s = UserChangedVote
			}
			log.Printf("User placed vote with value: \"%d\"", v)
			m.user.vote = v
			return nil, Message{T: StatusMsg, Value: s}
		}
		// Something went wrong
		return errors.New("placing vote error"), Message{}
	case StatusMsg:
		return nil, m
	default:
		return errors.New("unknown message type"), Message{}
	}
}

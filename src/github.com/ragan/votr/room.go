package votr

import (
	"fmt"
	"log"
	"time"

	"crypto/rand"
	"errors"
	"github.com/gorilla/websocket"
	"strconv"
)

// Every message must contain its type
type MessageType int

const (
	// Message occurs when user votes
	VoteMsg MessageType = 0
	// Message occurs when user enters or leaves room
	StatusMsg MessageType = 1
	// When message should be ignored by application
	IgnoreMsg MessageType = 2
	// Sent when user asks to reveal all users votes.
	// Room owner only.
	RevealMsg MessageType = 100
)

// Represents messages sent between users.
type Message struct {
	user      *User
	T         MessageType `json:"type"`
	Value     string      `json:"value,omitempty"`
	UserCount int         `json:"userCount"`
	VoteCount int         `json:"voteCount"`
	// Secret key for room administration
	Secret string `json:"secret,omitempty"`
}

var rooms = make(map[string]*Room)

type Room struct {
	users          map[*User]bool
	unregisterChan chan *User
	broadcastChan  chan Message
	secret         string
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
	s, err := randString(10)
	if err != nil {
		log.Printf("Error generating password: %s", err)
	}
	room := &Room{
		users:          make(map[*User]bool),
		unregisterChan: make(chan *User),
		broadcastChan:  make(chan Message),
		secret:         s,
	}
	rooms[id] = room
	go room.broadcast()
	return id
}

func randString(n int) (string, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz-"
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), nil
}

func GetIdLen() int {
	return 6
}

type RoomInfo struct {
	unregister chan *User
	broadcast  chan Message
	secret     string
}

// Adding user to room. Room should be created before adding user.
func addUser(u *User, roomId string) (error, RoomInfo) {
	log.Printf("Adding user to room \"%s\"", roomId)
	room, ok := rooms[roomId]
	if !ok {
		return fmt.Errorf("room \"%s\" does not exist", roomId), RoomInfo{}
	}
	secret := ""
	if len(room.users) == 0 {
		// this is the first user. send secret key
		secret = room.secret
	}
	room.users[u] = true
	log.Printf("Room users count: \"%v\"", len(room.users))
	return nil, RoomInfo{
		unregister: room.unregisterChan,
		broadcast:  room.broadcastChan,
		secret:     secret,
	}
}

func (r *Room) broadcast() {
	for {
		select {
		// New message
		case msg := <-r.broadcastChan:
			err, m := r.processMsg(msg)
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

// True when all users placed their vote
func (r *Room) done() bool {
	for u := range r.users {
		if u.vote == FirstVote {
			return false
		}
	}
	return true
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

func (r *Room) processMsg(m Message) (error, Message) {
	if m.T >= 100 {
		if r.secret != m.Secret {
			return fmt.Errorf("unprivileged function call with secret: \"%s\"", m.Secret), Message{T: IgnoreMsg}
		}
	}
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
	case RevealMsg:
		if !r.done() {
			return errors.New("voting not complete"), Message{}
		}
		s := ""
		c := 1
		for u := range r.users {
			s += strconv.Itoa(u.vote)
			if c != len(r.users) {
				s += ", "
			}
			c++
			// revert user vote value
			u.vote = FirstVote
		}
		return nil, Message{T: RevealMsg, Value: s}
	default:
		return errors.New("unknown message type"), Message{}
	}
}

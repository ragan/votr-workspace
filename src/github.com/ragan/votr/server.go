package votr

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	_ "net/http/pprof"
)

type User struct {
	conn     *websocket.Conn
	readChan chan []byte
	vote     int
}

type roomConn struct {
	ws     *websocket.Conn
	roomId string
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
		log.Printf("Root handler handling %s", r.URL)
		if r.URL.Path == "/" {
			id, ok := r.URL.Query()["r"]
			if !ok || len(id) < 1 {
				roomId := NewRoom()
				http.Redirect(w, r, "/?r="+roomId, http.StatusTemporaryRedirect)
			} else if !Exists(id[0]) {
				log.Printf("Room with id \"%s\" does not exist", id[0])
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			}
		}
		http.ServeFile(w, r, "static/index.html")
	}

func Go() {
	var upgrade = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", RootHandler)

	newConnection := make(chan roomConn)

	go serve(newConnection)

	http.HandleFunc("/ws", func(writer http.ResponseWriter,
		request *http.Request) {
		log.Printf("Incoming websocket request: %s", request.URL)
		conn, err := upgrade.Upgrade(writer, request, nil)
		if err != nil {
			log.Println(err)
			return
		}

		id, ok := request.URL.Query()["r"]
		if !ok || len(id) < 1 {
			log.Println("Got websocket request without \"r\" parameter.")
			return // todo: error message
		}

		log.Printf("Incoming websocket connection id=\"%s\"", id[0])
		newConnection <- roomConn{
			conn,
			id[0],
		}
	})

	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func serve(newCon chan roomConn) {
	for {
		select {
		case conn := <-newCon:
			// New connection
			log.Printf("New connection from %s", conn.ws.RemoteAddr())
			u := &User{
				conn:     conn.ws,
				readChan: make(chan []byte),
			}
			err, info := AddUser(u, conn.roomId)
			if err != nil {
				log.Print(err)
			}
			//users[u] = true
			//log.Printf("New User. Users count: %d", len(users))
			go u.read(info.newMessages, info.unregister)
			go u.write(conn.ws, u.readChan)
		}
	}
}

func (u *User) write(conn *websocket.Conn, read chan []byte) {
	ticker := time.NewTicker(tenSec)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		case msg := <-read:
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

const tenSec = 10 * time.Second

func (u *User) read(msgQ chan []byte, delete chan *User) {
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
		_, msg, err := u.conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			delete <- u
			break
		}
		msgQ <- msg
	}
}

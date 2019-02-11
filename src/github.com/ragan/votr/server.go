package votr

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"
)

const tenSec = 10 * time.Second

type votr struct {
	upg     websocket.Upgrader
	newConn chan roomConn
}

func (v votr) RootHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Root handler handling %s", r.URL)
	if r.URL.Path == "/" {
		id, ok := r.URL.Query()["r"]
		if !ok || len(id) < 1 {
			roomId := NewRoom()
			http.Redirect(w, r, "/?r="+roomId, http.StatusTemporaryRedirect)
		} else if _, ok := rooms[id[0]]; !ok {
			log.Printf("Room with id \"%s\" does not exist", id[0])
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}
	}
	http.ServeFile(w, r, "static/index.html")
}

func Go() {
	v := votr{
		upg: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		newConn: make(chan roomConn),
	}

	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", v.RootHandler)
	http.HandleFunc("/ws", v.WSHandler)

	go v.serve()

	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func (v votr) WSHandler(writer http.ResponseWriter, request *http.Request) {
	// https://gist.github.com/tmichel/7390690#file-ws-go-L31
	log.Printf("Incoming websocket request: %s", request.URL)
	conn, err := v.upg.Upgrade(writer, request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	id, ok := request.URL.Query()["r"]
	if !ok || len(id) < 1 {
		log.Println("Got websocket request without \"r\" parameter.")
		return
	}

	log.Printf("Incoming websocket connection id=\"%s\"", id[0])
	v.newConn <- roomConn{
		ws:     conn,
		roomId: id[0],
	}
}

func (v votr) serve() {
	for {
		select {
		// New connection
		case newConn := <-v.newConn:
			log.Printf("New connection from %s", newConn.ws.RemoteAddr())
			u := NewUser(newConn.ws)
			// Room should be created by now
			err, info := addUser(u, newConn.roomId)
			if err != nil {
				log.Print(err)
				break
			}
			go u.read(info.broadcast, info.unregister)
			go u.write(newConn.ws)

			info.broadcast <- Message{
				T:     StatusMsg,
				Value: UserEnteredMsg,
				user:  u,
			}
		}
	}
}

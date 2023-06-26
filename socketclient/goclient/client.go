package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	SERVER_HOST = "localhost"
	SERVER_PORT = ":9898" // literal port number or service name.
	SERVER_TYPE = "tcp"
)

/*
https://medium.com/swlh/handle-concurrency-in-gorilla-web-sockets-ade4d06acd9c
*/
var addr = flag.String("addr", SERVER_HOST+SERVER_PORT, "http service address")

func formatCommand(s1, s2 string, width int) {
	fmt.Println(s1 + strings.Repeat(" ", width-len(s1)) + ": " + s2)
}

func printCommands() {
	const dist1 = 45
	fmt.Println()
	formatCommand("Available case-sensitive commands", "", dist1)
	formatCommand(" login <myName>", "login to the streaming data server", dist1)
	formatCommand(" groups", "list available time series groups (more than 377917)", dist1)
	formatCommand(" group.device <group>", "list available time series devices for a specific group. Example: group.device synthetic", dist1)
	formatCommand(" timeseries <group.device>", "list the measurement names for a specifc group & device. Example: timeseries synthetic.IoT_Weather", dist1)
	formatCommand(" count <group.device>", "count the number of measurement values for a specifc group & device. Example: count synthetic.IoT_Weather", dist1)
	formatCommand(" data <group.device> interval 1s format csv", "stream the data for a specifc group & device at 1 second intervals in CSV format. Example: data synthetic.IoT_Weather interval 1s format csv", dist1)
	formatCommand(" data <group.device> interval 5s format json", "stream the data for a specifc group & device at 5 second intervals in JSON format. Example: data synthetic.IoT_Weather interval 5s format json", dist1)
	formatCommand(" logout", "stop sending me data; I have a headache", dist1)
	fmt.Println()
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	printCommands()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Close the connection: send a close message, then wait for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

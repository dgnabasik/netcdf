// Socket client for golang: make sure local firewall isn’t blocking.
// Golang allows for 2 socket types:
//  1. Unix domain sockets (AF_UNIX); on same machine for interprocess communication; only the kernel is involved in the communication between processes; since domain sockets are files, the file's permissions determine who can send data to the socket.
//  2. Network sockets (AF_INET|AF_INET6).
//
// Using an HTTP server with a Unix domain socket in Go can provide several advantages such as improved security, performance, ease of use, and interoperability.
// Two kinds of unix sockets: stream (tcp) & datagram(udp; discrete data chunks).
// What about compression, encryption, authentication.?
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

const (
	SERVER_HOST = "localhost"
	SERVER_PORT = "9988" // literal port number or service name.
	SERVER_TYPE = "tcp"
)

func main() {
	conn, err := net.Dial(SERVER_TYPE, SERVER_HOST+":"+SERVER_PORT)
	if err != nil {
		panic(err)
	}
	_, err = connection.Write([]byte("Hello Server! Greetings."))
	buffer := make([]byte, 1024)
	mLen, err := connection.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	fmt.Println("Received: ", string(buffer[:mLen]))
	defer connection.Close()
}

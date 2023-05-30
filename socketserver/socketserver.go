// The traditional safeguard is to establish multiple DNS servers for a given site where each DNS client on the network is configured with each of those servers' IP addresses.
// An anycast address is an address allocated to a set of interfaces that typically belong to different routers. When a packet is destined to an anycast address, it is delivered to the "closest" (i.e., routing protocol) interface that has this anycast address.
// Anycast is the concept of taking one IP address and sharing it between multiple servers, each unaware of the others. DNS redundancy and failover is a classic use case for anycast.
// Anycast is not limited to DNS. It can be used to provide redundancy and failover for any number of stateless protocols and applications. https://www.linuxjournal.com/magazine/ipv4-anycast-linux-and-quagga
// Your anycast address should be on its own subnet, separate from any other existing subnets. The anycast subnet must never be included in a summary.
// An anycast address must be assigned to a router not a host and cannot be used as a source address.
// 3 types of IPv6 addresses: Unicast, Anycast, Multicast. https://kasiviswanathanblog.wordpress.com/2017/03/18/ipv6-address-types-and-the-rest/
// Create a separate Goroutine to serve each TCP client.  Execute netcat to test: nc 127.0.0.1 <port>
// lsof -Pnl +M -i4	or -i6		netstat -tulpn
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	SERVER_HOST = "localhost" // if not specified, the function listens on all available unicast and anycast IP addresses of the local system.
	SERVER_PORT = "989898"
	SERVER_TYPE = "tcp4" // tcp, tcp4, tcp6, unix, or unix packet
)

var count = 0

func handleConnection(c net.Conn) {
	fmt.Print(".")
	for {
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		temp := strings.TrimSpace(string(netData))
		if temp == "STOP" {
			break
		}
		fmt.Println(temp)
		counter := strconv.Itoa(count) + "\n"
		c.Write([]byte(string(counter)))
	}
	c.Close()
}

func main() {
	fmt.Println("Server Running...")
	server, err := net.Listen(SERVER_TYPE, SERVER_HOST+":"+SERVER_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer server.Close()
	fmt.Println("Listening on " + SERVER_HOST + ":" + SERVER_PORT)
	for {
		connection, err := server.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(connection)
		count++
	}
}

package main

import (
	"fmt"
	"net"
	"net/netip"
	"os/exec"

	"github.com/k0kubun/pp"

	a "github.com/seancfoley/ipaddress-go/ipaddr"
)

func GetIPrange() {
	var addrs []*a.IPAddress
	block := a.NewIPAddressString("8.8.8.0/24").GetAddress()
	for i := block.Iterator(); i.HasNext(); {
		addrs = append(addrs, i.Next())
	}
	fmt.Println(len(addrs)) // 8.8.8.0/24 through 8.8.8.255/24
}

func Hosts(cidr string) ([]netip.Addr, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		panic(err)
	}

	var ips []netip.Addr
	for addr := prefix.Addr(); prefix.Contains(addr); addr = addr.Next() {
		ips = append(ips, addr)
	}

	if len(ips) < 2 {
		return ips, nil
	}

	return ips[1 : len(ips)-1], nil
}

// http://play.golang.org/p/m8TNTtygK0
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

type Pong struct {
	Ip    string
	Alive bool
}

func ping(pingChan <-chan string, pongChan chan<- Pong) {
	for ip := range pingChan {
		_, err := exec.Command("ping", "-c1", "-t1", ip).Output()
		var alive bool
		if err != nil {
			alive = false
		} else {
			alive = true
		}
		pongChan <- Pong{Ip: ip, Alive: alive}
	}
}

func receivePong(pongNum int, pongChan <-chan Pong, doneChan chan<- []Pong) {
	var alives []Pong
	for i := 0; i < pongNum; i++ {
		pong := <-pongChan
		//  fmt.Println("received:", pong)
		if pong.Alive {
			alives = append(alives, pong)
		}
	}
	doneChan <- alives
}

func main() {
	hosts, _ := Hosts("192.168.1.47/24")
	concurrentMax := 100
	pingChan := make(chan string, concurrentMax)
	pongChan := make(chan Pong, len(hosts))
	doneChan := make(chan []Pong)

	for i := 0; i < concurrentMax; i++ {
		go ping(pingChan, pongChan)
	}

	go receivePong(len(hosts), pongChan, doneChan)

	for _, ip := range hosts {
		pingChan <- ip.String()
		//  fmt.Println("sent: " + ip)
	}

	alives := <-doneChan
	pp.Println(alives)
}

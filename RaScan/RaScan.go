package main

import (
	"flag"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// var webS = `
// GET / HTTP/1.1
// Host: ø
// Connection: close
// `
// var subnet string
var Red = "\033[31;1m"
var Green = "\033[32;1m"

//var Reset = "\033[0m"

const (
	ProtocolICMP = 1
)

func main() {
	showAll := flag.Bool("sA", false, "To show all available and unavaible IPs.")
	timeout := flag.Int("o", 3000, "Set timeout for each IP.")
	flag.Parse()

	full := flag.Args()
	file := full[1]

	if *showAll {
		_ = fmt.Sprint(timeout)
	}

	fil, err := os.Create(file)
	if err != nil {
		fmt.Println(err)
	}
	defer fil.Close()

	Smask, err := strconv.Atoi(strings.Split(full[0], "/")[1])
	if err != nil {
		fmt.Println(err)
	}
	Us := 32 - Smask
	usable := math.Pow(2, float64(Us))
	thisIP := strings.Split(full[0], "/")[0]

	a := strings.Split(thisIP, ".")
	for i := 1; i < int(usable); i++ {
		a[3] = strconv.Itoa(i)
		addr := strings.Join(a, ".")
		dst, dur, hn, err := Ping(addr, time.Duration(*timeout))
		if err != nil {
			if *showAll {
				fmt.Printf("%v[-] Dead connection %v: %v      (%v)\n", Red, dst, err, dur)
				continue
			}
		} else {
			if *showAll {
				fmt.Printf("%v[+] Alive connection %v (%v) %v\n", Green, dst, hn, dur)
				fil.Write([]byte(fmt.Sprintf("[+] Alive connection %v (%v) %v\n", dst, hn, dur)))
				continue
			}
		}
	}
}

var ListenAddr = "0.0.0.0"

func Ping(addr string, timeout time.Duration) (*net.IPAddr, time.Duration, string, error) {
	c, err := icmp.ListenPacket("ip4:icmp", ListenAddr)
	if err != nil {
		return nil, 0, "", err
	}
	defer c.Close()

	dst, err := net.ResolveIPAddr("ip4", addr)
	if err != nil {
		return nil, 0, "", err
	}

	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte(""),
		},
	}

	b, err := m.Marshal(nil)
	if err != nil {
		return dst, 0, "", err
	}

	start := time.Now()
	n, err := c.WriteTo(b, dst)
	if err != nil {
		return dst, 0, "", err
	} else if n != len(b) {
		return dst, 0, "", fmt.Errorf("got: %v, wants: %v", n, len(b))
	}

	reply := make([]byte, 1500)
	err = c.SetReadDeadline(time.Now().Add(timeout * time.Millisecond))
	if err != nil {
		return dst, 2, "", fmt.Errorf("ping: %v timeout", dst)
	}
	n, peer, err := c.ReadFrom(reply)
	if err != nil {
		return dst, 0, "", err
	}
	duration := time.Since(start)

	rm, err := icmp.ParseMessage(ProtocolICMP, reply[:n])
	if err != nil {
		return dst, 0, "", err
	}
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		hname, err := net.LookupAddr(addr)
		if err != nil {
			hname = append(hname, fmt.Sprintf("%v", err))
		}
		return dst, duration, hname[0], nil
	default:
		return dst, 0, "", fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
	}
}

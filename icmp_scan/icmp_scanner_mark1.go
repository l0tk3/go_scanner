package icmp_scan

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go_scanner/tools"
	"net"
	"sync"
	"time"
)

type ICMP struct {
	Type        uint8
	Code        uint8
	Checksum    uint16
	Identifier  uint16
	SequenceNum uint16
}

var alive_list []string

func pack_icmp_echo_request(ident, seq uint16, payload []byte) []byte {
	icmp := ICMP{8, 0, 0, ident, seq}
	var buffer bytes.Buffer
	binary.Write(&buffer, binary.BigEndian, icmp)
	binary.Write(&buffer, binary.BigEndian, payload)
	b := buffer.Bytes()
	binary.BigEndian.PutUint16(b[2:], uint16(tools.Calculate_checksum(b)))
	return b
}

func unpack_icmp_echo_reply(icmpBytes []byte) ICMP {
	var icmp ICMP
	buffer := bytes.NewBuffer(icmpBytes)
	binary.Read(buffer, binary.BigEndian, &icmp)
	return icmp
}

func send_icmp(ip, seq int) bool {

	current_ip := tools.Int2ip(int32(ip))
	ident := uint16(ip)
	icmp_pack := Make_icmp_pack(ident)
	conn, err := net.DialTimeout("ip4:icmp", current_ip, 1*time.Second)
	if err != nil {
		return false
	}
	if _, err := conn.Write(icmp_pack); err != nil {
		return false
	}
	conn.SetReadDeadline((time.Now().Add(time.Second * 6)))
	recv := make([]byte, 60)
	len, err := conn.Read(recv)
	conn.Close()
	if err != nil {
		return false
	}
	recv = recv[20:len]
	recv_icmp := unpack_icmp_echo_reply(recv[0:8])
	if recv_icmp.Type != 0 || recv_icmp.Code != 0 {
		return false
	}
	if recv_icmp.Identifier != ident {
		return false
	}
	// fmt.Println(current_ip, " is alive")
	return true
}

func Make_icmp_pack(ident uint16) []byte {
	magic := []byte("nihaoa")
	sending_ts := time.Now().Unix()
	var time_stamp []byte = make([]byte, 8)
	binary.BigEndian.PutUint64(time_stamp, uint64(sending_ts))
	payload := time_stamp
	payload = append(payload, magic...)
	icmp_pack := pack_icmp_echo_request(ident, uint16(1), payload)
	return icmp_pack
}
func Icmp_scan1(start, end int) []string {
	var wg sync.WaitGroup
	var sub_wg sync.WaitGroup
	rate := time.Second / 1000
	throttle := time.Tick(rate)
	ip_chan := make(chan int, 65536)
	alive_chan := make(chan string, 65536)
	thread_num := 1000
	for i := 0; i < thread_num; i++ {
		go func() {
			for current_ip := range ip_chan {
				// fmt.Println(current_ip)
				if send_icmp(current_ip, i%65536) {
					sub_wg.Add(1)
					alive_chan <- tools.Int2ip(int32(current_ip))
					fmt.Println("Host:", tools.Int2ip(int32(current_ip)), "is alive")
				}
				wg.Done()
				<-throttle
			}

		}()
	}

	go func() {
		for alive := range alive_chan {
			alive_list = append(alive_list, alive)
			sub_wg.Done()
		}
	}()

	for i := start; i <= end; i++ {
		wg.Add(1)
		ip_chan <- i
	}
	close(ip_chan)
	wg.Wait()
	sub_wg.Wait()
	close(alive_chan)
	return alive_list
}

package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

var id uint16
var typeCodes = map[string]uint16{
	"A":     1,
	"NS":    2,
	"CNAME": 5,
	"MX":    15,
	"TXT":   16,
	"AAAA":  28,
}
var typeNames = map[uint16]string{
	1:  "A",
	2:  "NS",
	5:  "CNAME",
	15: "MX",
	16: "TXT",
	28: "AAAA",
}

func createDNSMessage(address string, tp string) (uint16, []byte, error) {
	var message []byte
	var header []byte
	var question []byte

	id += uint16(1)

	messageID := id

	header = append(header, byte(messageID>>8), byte(messageID))
	header = append(header, byte(1), byte(0), byte(0), byte(1), byte(0), byte(0), byte(0), byte(0), byte(0), byte(0))
	
	sectionsUrl := strings.Split(address, ".")
	for _, sec := range sectionsUrl {
		url_bytes := []byte(sec)
		question = append(question, byte(len(url_bytes)))
		question = append(question, url_bytes...)
	}
	question = append(question, byte(0))

	qtype, found := typeCodes[tp]
	if found == false {
		return 0, []byte{}, fmt.Errorf("Тип не поддерживается")
	}
	question = append(question, byte(qtype>>8), byte(qtype))
	question = append(question, byte(0), byte(1))

	message = append(message, header...)
	message = append(message, question...)

	return messageID, message, nil
}

func sendDNSMessage(serverAddress string, message []byte) ([]byte, error) {
	udpAddress, err := net.ResolveUDPAddr("udp", serverAddress)
	if err != nil {
		return nil, fmt.Errorf("Неверный адрес для отправки")
	}

	conn, err := net.DialUDP("udp", nil, udpAddress)
	if err != nil {
        return nil, fmt.Errorf("connect UDP: %w", err)
    }
	defer conn.Close()

	duration := time.Now().Add(5 * time.Second)
	err = conn.SetDeadline(duration)
	_, err = conn.Write(message)
    if err != nil {
		return nil, fmt.Errorf("Ошибка отправки: %w", err)
    }
	
	response := make([]byte, 4096)

	n, err := conn.Read(response)
	if err != nil {
		return nil, fmt.Errorf("Ответ %w", err)
	}

	response = response[:n]
	return response, nil
}

func uint32IPtoString(address uint32) string {
	first := uint8(address >> 24)
	second := uint8(address >> 16)
	third := uint8(address >> 8)
	fourth := uint8(address)

	return fmt.Sprintf(
		"%d.%d.%d.%d",
		first,
		second,
		third,
		fourth,
	)
}

func readDNSName(message []byte, start int) (string, int, error) {
	
}

func parseDNSResponse(messageID uint16, response []byte) (string, byte, error){
	if messageID != binary.BigEndian.Uint16(response[0:2]) && response[2]>>7 == byte(1){
		return "", 0, fmt.Errorf("Неверный ID ответа")
	}
	rcode := (response[3]<<4)>>4

	headerInd := 12

	qEnd := readDNSName(response, headerInd)
	
	tp := binary.BigEndian.Uint16(response[qEnd+2:qEnd+4])
	typeName := typeNames[tp]

	// ttl := binary.BigEndian.Uint32(response[qEnd+6:qEnd+8])

	rdlength := binary.BigEndian.Uint16(response[qEnd+10:qEnd+12])
	rdataStart := qEnd + 12
	rdataEnd := rdataStart + int(rdlength)
	if rdataEnd > len(response) {
		return "", 0, fmt.Errorf("RDATA exceeds response size")
	}
	rdata := response[rdataStart:rdataEnd]

	if typeName == "A" {
		ipv4Uint := binary.BigEndian.Uint32(rdata)
		ipv4 := uint32IPtoString(ipv4Uint)
		return ipv4, rcode, nil
	}
	return "", 0, nil
} 

func resolveDNS(serverAddress string, address string, tp string) (uint32, error) {
	messageID, message, err := createDNSMessage(address, tp)
	if err != nil {
		return uint32(0), fmt.Errorf("Error: %w", err)
	}

	resp, err := sendDNSMessage(serverAddress, message)
	if err != nil {
		return uint32(0), fmt.Errorf("Error: %w", err)
	}

	ip, rcode, err := parseDNSResponse(messageID, resp)
	if err != nil {
		return 0, fmt.Errorf("Erorr %w", err)
	}

	fmt.Println(ip)
	fmt.Println(rcode)
	return uint32(0), nil
}

func main() {
	serverAddress := net.JoinHostPort(os.Args[1], os.Args[2])

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		
		address, tp, found := strings.Cut(line, " ")
		if !found {
			continue
		}
		_, err := resolveDNS(serverAddress, address, tp)
		if err != nil {
			fmt.Println("Erorr", err)
		}
	}
}
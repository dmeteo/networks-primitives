package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
)

type EthernetHeader struct {
	dst       string
	src       string
	etherType uint16
}

type IPv4Header struct {
    version       uint8
	ihlBytes      int
    totalLength   uint16
    id            uint16
    flags         uint16
    fragOffset    uint16
    ttl           uint16
    protocol      uint16
    src           string
    dst           string
    checksumValid bool
}

type TCPHeader struct {
	srcPort uint16
	dstPort uint16
	seq uint32
	ack uint32
	dataOffsetBytes uint16
	flags string
	window uint16
}

type UDPHeader struct {
	srcPort uint16
	dstPort uint16
	length uint16
}


func readPacket() []byte {
	var packetBytes []byte
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "exit" {
			break
		}

		bytes, err := hex.DecodeString(line)
		if err != nil {
			fmt.Print("Ошибка")
			continue
		}

		packetBytes = append(packetBytes, bytes...)
	}

	return packetBytes
}

func parseMacAddress(bytes []byte) string {
	var builder strings.Builder

	for i, val := range bytes[0:6] {
		if i > 0 {
			builder.WriteString(":")
		}
		fmt.Fprintf(&builder, "%02x", val)
	}

	return builder.String()
}

func parseEthernetLayer(frame []byte) EthernetHeader {
	var ethernetHeader EthernetHeader
	ethernetHeader.dst = parseMacAddress(frame[0:6])
	ethernetHeader.src = parseMacAddress(frame[6:12])
	ethernetHeader.etherType = binary.BigEndian.Uint16(frame[12:(6 + 6 + 2)])

	return ethernetHeader
}

func printEthernetLayer(ethernetHeader *EthernetHeader) {
	fmt.Println("eth.dst", ethernetHeader.dst)
	fmt.Println("eth.src", ethernetHeader.src)
	fmt.Printf("eth.ethertype 0x%04x\n", ethernetHeader.etherType)
}

// func ipv4ChecksumValid(header []byte) bool {

// }

func parseIPv4Layer(frame []byte) IPv4Header {
	var IPv4Header IPv4Header

	firstByte := frame[0]
	IPv4Header.version = uint8(firstByte >> 4)
	IPv4Header.ihlBytes = int((firstByte << 4) >> 4) * 4
	IPv4Header.totalLength = binary.BigEndian.Uint16(frame[2:4])
	IPv4Header.id = binary.BigEndian.Uint16(frame[4:6])
	flagsAndOffset := binary.BigEndian.Uint16(frame[6:8])
	IPv4Header.flags = uint16(flagsAndOffset >> 13)
	IPv4Header.fragOffset = uint16((flagsAndOffset << 3) >> 3)
	IPv4Header.ttl = uint16(frame[8])
	IPv4Header.protocol = uint16(frame[9])
	IPv4Header.src = net.IP(frame[12:16]).String()
	IPv4Header.dst = net.IP(frame[16:20]).String()
	// checksum := frame[10:12]

	return IPv4Header
}  

func main() {
	packetBytes := readPacket()
	ethernetHeaderEnd := 14
	ethernetHeader := parseEthernetLayer(packetBytes[:ethernetHeaderEnd])
	printEthernetLayer(&ethernetHeader)

	if ethernetHeader.etherType == 0x0800 {
		// IPv4Header := parseIPv4Layer(packetBytes[ethernetHeaderEnd:])
	}

}

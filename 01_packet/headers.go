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
	srcPort         uint16
	dstPort         uint16
	seq             uint32
	ack             uint32
	dataOffsetBytes uint16
	flags           string
	window          uint16
}

type UDPHeader struct {
	srcPort uint16
	dstPort uint16
	length  uint16
}

var tcpFlags = []struct {
	mask byte
	name string
}{
	{0x01, "FIN"},
	{0x02, "SYN"},
	{0x04, "RST"},
	{0x08, "PSH"},
	{0x10, "ACK"},
	{0x20, "URG"},
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

func formatIPv4Flags(flags uint16) string {
	names := []string{}

	if flags&0b010 != 0 {
		names = append(names, "DF")
	}
	if flags&0b001 != 0 {
		names = append(names, "MF")
	}

	if len(names) == 0 {
		return "none"
	}

	return strings.Join(names, ",")
}

func printIPv4Layer(header *IPv4Header) {
	fmt.Println("ip.version", header.version)
	fmt.Println("ip.ihl_bytes", header.ihlBytes)
	fmt.Println("ip.total_length", header.totalLength)
	fmt.Printf("ip.id 0x%04x\n", header.id)
	fmt.Println("ip.flags", formatIPv4Flags(header.flags))
	fmt.Println("ip.frag_offset", header.fragOffset)
	fmt.Println("ip.ttl", header.ttl)
	fmt.Println("ip.protocol", header.protocol)
	fmt.Println("ip.src", header.src)
	fmt.Println("ip.dst", header.dst)
	fmt.Println("ip.checksum_valid", header.checksumValid)
}

func printTCPLayer(header *TCPHeader) {
	fmt.Println("tcp.src_port", header.srcPort)
	fmt.Println("tcp.dst_port", header.dstPort)
	fmt.Println("tcp.seq", header.seq)
	fmt.Println("tcp.ack", header.ack)
	fmt.Println("tcp.data_offset_bytes", header.dataOffsetBytes)
	fmt.Println("tcp.flags", header.flags)
	fmt.Println("tcp.window", header.window)
}

func printUDPLayer(header *UDPHeader) {
	fmt.Println("udp.src_port", header.srcPort)
	fmt.Println("udp.dst_port", header.dstPort)
	fmt.Println("udp.length", header.length)
}

func printPayloadLength(length int) {
	fmt.Println("payload.length", length)
}

func ipv4ChecksumValid(header []byte) bool {
	checksum := binary.BigEndian.Uint16(header[10:12])

	var sum uint32

	for i := 0; i < len(header); i += 2 {
		if i == 10 {
			continue
		}
		val := binary.BigEndian.Uint16(header[i : i+2])
		sum += uint32(val)
	}

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	calculatedChecksum := ^uint16(sum)

	return calculatedChecksum == checksum
}

func parseIPv4Layer(frame []byte) IPv4Header {
	var IPv4Header IPv4Header

	firstByte := frame[0]
	IPv4Header.version = uint8(firstByte >> 4)
	IPv4Header.ihlBytes = int((firstByte<<4)>>4) * 4
	IPv4Header.totalLength = binary.BigEndian.Uint16(frame[2:4])
	IPv4Header.id = binary.BigEndian.Uint16(frame[4:6])
	flagsAndOffset := binary.BigEndian.Uint16(frame[6:8])
	IPv4Header.flags = uint16(flagsAndOffset >> 13)
	IPv4Header.fragOffset = uint16((flagsAndOffset<<3)>>3) * 8
	IPv4Header.ttl = uint16(frame[8])
	IPv4Header.protocol = uint16(frame[9])
	IPv4Header.src = net.IP(frame[12:16]).String()
	IPv4Header.dst = net.IP(frame[16:20]).String()
	IPv4Header.checksumValid = ipv4ChecksumValid(frame[:IPv4Header.ihlBytes])

	return IPv4Header
}

func formatTCPFlags(value byte) string {
	names := []string{}

	for _, flag := range tcpFlags {
		if value&flag.mask != 0 {
			names = append(names, flag.name)
		}
	}
	if len(names) == 0 {
		return "none"
	}

	return strings.Join(names, ",")
}

func parseTCPLayer(frame []byte) TCPHeader {
	var TCPHeader TCPHeader

	TCPHeader.srcPort = binary.BigEndian.Uint16(frame[0:2])
	TCPHeader.dstPort = binary.BigEndian.Uint16(frame[2:4])
	TCPHeader.seq = binary.BigEndian.Uint32(frame[4:8])
	TCPHeader.ack = binary.BigEndian.Uint32(frame[8:12])
	dof := binary.BigEndian.Uint16(frame[12:14])
	TCPHeader.dataOffsetBytes = (dof >> 12) * 4
	flagsByte := frame[13]
	TCPHeader.flags = formatTCPFlags(flagsByte)
	TCPHeader.window = binary.BigEndian.Uint16(frame[14:16])

	return TCPHeader
}

func parseUDPLayer(frame []byte) UDPHeader {
	var UDPHeader UDPHeader

	UDPHeader.srcPort = binary.BigEndian.Uint16(frame[0:2])
	UDPHeader.dstPort = binary.BigEndian.Uint16(frame[2:4])
	UDPHeader.length = binary.BigEndian.Uint16(frame[4:6])

	return UDPHeader
}

func main() {
	var length int
	packetBytes := readPacket()
	ethernetHeaderEnd := 14
	ethernetHeader := parseEthernetLayer(packetBytes[:ethernetHeaderEnd])
	printEthernetLayer(&ethernetHeader)

	if ethernetHeader.etherType == 0x0800 {
		IPv4Header := parseIPv4Layer(packetBytes[ethernetHeaderEnd:])
		printIPv4Layer(&IPv4Header)

		if IPv4Header.protocol == 6 {
			TCPHeader := parseTCPLayer(packetBytes[ethernetHeaderEnd+IPv4Header.ihlBytes:])
			length = int(TCPHeader.dataOffsetBytes)
			printTCPLayer(&TCPHeader)
		}
		if IPv4Header.protocol == 17 {
			UDPHeader := parseUDPLayer(packetBytes[ethernetHeaderEnd+IPv4Header.ihlBytes:])
			length = 8
			printUDPLayer(&UDPHeader)
		}
		payloadLength := int(IPv4Header.totalLength) - IPv4Header.ihlBytes - length
		printPayloadLength(payloadLength)
	}
}

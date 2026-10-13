package main

import (
	"bufio"
	"net"
	"os"
	"strings"
)

func main() {
	serverAddress := net.JoinHostPort(os.Args[0], os.Args[1])

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		
		address, tp, found := strings.Cut(line, " ")
		if !found {
			continue
		}
	}
}
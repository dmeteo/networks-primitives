package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Subnet struct {
	network      uint32
	broadcast    uint32
	netmask      uint32
	prefix       uint8
	first        uint32
	last         uint32
	hosts        uint64
	hasBroadcast bool
}

func parseSubnet(addressPrefix string) ([]string, error) {
	addressStr, prefixStr, found := strings.Cut(addressPrefix, "/")
	if found == false {
		return []string{}, fmt.Errorf("Invalid subnet")
	}
	return []string{addressStr, prefixStr}, nil
}

func parsePrefix(prefixStr string) (uint8, error) {
	prefix, err := strconv.Atoi(prefixStr)
	if err != nil {
		return 33, fmt.Errorf("Invalid prefix %w", err)
	}
	return uint8(prefix), nil
}

func parseIPv4(addressStr string) (uint32, error) {
	address := strings.Split(addressStr, ".")

	okt1, err := strconv.Atoi(address[0])
	okt2, err := strconv.Atoi(address[1])
	okt3, err := strconv.Atoi(address[2])
	okt4, err := strconv.Atoi(address[3])
	if err != nil {
		return uint32(0), fmt.Errorf("invalid address")
	}

	ip := uint32(okt1) << 24 + uint32(okt2) << 16 + uint32(okt3) << 8 + uint32(okt4)

	return ip, nil
}

func subnetAnalyzer(addressPrefix string) (Subnet, error) {
	var subnet Subnet

	addressPref, err := parseSubnet(addressPrefix)
	if err != nil {
		return Subnet{}, fmt.Errorf("invalid address")
	}

	addressStr := addressPref[0]
	prefixStr := addressPref[1]

	ip, err := parseIPv4(addressStr)
	prefix, err := parsePrefix(prefixStr)
	subnet.prefix = prefix

	hostBits := 32 - int(prefix)

	network := ip
	broadcast := ip

	for bit := 0; bit < hostBits; bit++ {
		network = network & ^(uint32(1) << bit)
		broadcast = broadcast | uint32(1) << bit
	}
	subnet.broadcast = broadcast
	subnet.network = network

	subnet.netmask = (^uint32(0) >> hostBits) << hostBits
	
	switch subnet.prefix {
	case 32:
		subnet.first = subnet.network
		subnet.last = subnet.network
		subnet.hosts = 1
		subnet.hasBroadcast = false
	case 31:
		subnet.first = subnet.network
		subnet.last = subnet.network + 1
		subnet.hosts = 2
		subnet.hasBroadcast = false
	default:
		subnet.hasBroadcast = true
		subnet.first = subnet.network + 1
		subnet.last = subnet.broadcast - 1
		subnet.hosts = (uint64(1) << hostBits) - 2
	}

	return subnet, nil
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

func printSubnet(subnet *Subnet) {
	fmt.Println("network", uint32IPtoString(subnet.network))
	if subnet.hasBroadcast {
		fmt.Println("broadcast", uint32IPtoString(subnet.broadcast))
	} else {
		fmt.Println("broadcast none")
	}
	fmt.Println("netmask", uint32IPtoString(subnet.netmask))
	fmt.Println("prefix", subnet.prefix)
	fmt.Println("first", uint32IPtoString(subnet.first))
	fmt.Println("last", uint32IPtoString(subnet.last))
	fmt.Println("hosts", subnet.hosts)
}

type Route struct {
	network uint32
	netmask uint32
	prefix uint8
	interfaceName string
}

func readRouterTable(file *os.File) ([]Route, error) {
	var routeTable []Route

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("invalid route line %q", line)
		}
		addressPrefix := fields[0]
		interfaceName := fields[1]
		addressPref, err := parseSubnet(addressPrefix)
		if err != nil {
			return []Route{}, fmt.Errorf("invalid subnet")
		}
		addressStr := addressPref[0]
		prefixStr := addressPref[1]

		network, err := parseIPv4(addressStr)
		if err != nil {
			return nil, err
		}

		prefix, err := parsePrefix(prefixStr)
		if err != nil {
			return nil, err
		}

		hostBits := 32 - int(prefix)
		netmask := (^uint32(0) >> hostBits) << hostBits
		network &= netmask
		routeTable = append(routeTable, Route{network: network, netmask: netmask, prefix: prefix, interfaceName: interfaceName})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read route table: %w", err)
	}

	return routeTable, nil
}

func selectRoute(routes []Route, dst uint32) (Route, bool) {
	var bestRoute Route
	found := false

	for _, route := range routes {
		if dst&route.netmask == route.network {
			if !found || route.prefix > bestRoute.prefix {
				bestRoute = route
				found = true
			}
		}
	}
	
	return bestRoute, found
}

func routeSelector(pathTable string, dst string) (Route, error) {
	file, err := os.Open(pathTable)

	if err != nil {
		return Route{}, fmt.Errorf("Ошибка открытия файла %w", err)
	}
	defer file.Close()

	routeTable, err := readRouterTable(file)
	if err != nil {
		return Route{}, fmt.Errorf("Ошибка чтения таблицы %w", err)
	}
	destination, err := parseIPv4(dst)
	if err != nil {
		return Route{}, fmt.Errorf("Ошибка адреса назначения %w", err)
	}

	bestRoute, found := selectRoute(routeTable, destination)
	if found == false {
		return Route{}, fmt.Errorf("Адрес не найден %w", err)
	}
	return bestRoute, nil
}

func printSelectedRoute(route *Route) {
	fmt.Println("via", route.interfaceName)
	fmt.Println("prefix", route.prefix)
}

func main() {
	if os.Args[1] == "subnet" {
		subnet, err := subnetAnalyzer(os.Args[2])
		if err != nil {
			fmt.Errorf("Invalid subnet %w", err)
		}
		printSubnet(&subnet)
	}
	if os.Args[1] == "route" {
		route, err := routeSelector(os.Args[2], os.Args[3])
		if err != nil {
			fmt.Println("unreachable true")
			os.Exit(1)
		}
		printSelectedRoute(&route)
	}
}
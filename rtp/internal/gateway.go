package internal

import (
	"log"
	"net"
	"reflect"
	"strings"
	"syscall"
)

func GetLocalAddresses() []string {
	names := []string{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return names
	}
	for _, i := range ifaces {
		// check that it has a valid gateway address.
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		if strings.HasPrefix(i.Name, "usb") || strings.HasPrefix(i.Name, "eth") || strings.HasPrefix(i.Name, "wlan") {
			hasIPv4 := false
			for _, a := range addrs {
				switch v := a.(type) {
				case *net.IPNet:
					if v.IP.To4() != nil {
						hasIPv4 = true
					}
				}
			}
			if hasIPv4 {
				names = append(names, i.Name)
			}
		}
	}
	return names
}

func bindToDevice(conn net.UDPConn, device string) error {
	ptrVal := reflect.ValueOf(conn)
	fdmember := reflect.Indirect(ptrVal).FieldByName("fd")
	pfdmember := reflect.Indirect(fdmember).FieldByName("pfd")
	netfdmember := reflect.Indirect(pfdmember).FieldByName("Sysfd")
	fd := int(netfdmember.Int())
	return syscall.SetsockoptString(fd, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, device)
}

func DialVia(to *net.UDPAddr, via string) (*net.UDPConn, error) {
	conn, err := net.DialUDP("udp", nil, to)
	if err != nil {
		log.Printf("error dialing to %v %v", to, err)
		return nil, err
	}
	if err := bindToDevice(*conn, via); err != nil {
		log.Printf("error binding to device %v %v", via, err)
		return nil, err
	}
	return conn, nil
}

package utils

import "net"

func CheckUDPPortAvailability(port uint32) error {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{Port: int(port)})
	if err != nil {
		return err
	}
	_ = ln.Close()
	return nil
}

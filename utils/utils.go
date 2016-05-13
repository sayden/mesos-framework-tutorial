package utils

import (
	"log"
	"net"

	"github.com/sayden/mesos-framework-tutorial/server"
)

func ParseIP(address string) net.IP {
	addr, err := net.LookupIP(address)
	if err != nil {
		log.Fatal(err)
	}
	if len(addr) < 1 {
		log.Fatalf("failed to parse IP from address '%v'", address)
	}
	return addr[0]
}

// getExecutorCmd takes a full local path like /tmp/something/file and returns
// the last element as a bash executable command => ./file
func GetExecutorCmd(path string) string {
	return "." + server.GetHttpPath(path)
}

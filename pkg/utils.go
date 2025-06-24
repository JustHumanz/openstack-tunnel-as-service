package pkg

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"time"
)

const ipv4Regex = `\b(?:\d{1,3}\.){3}\d{1,3}\b`
const tcpTimeout = 3 * time.Second

func FindVMactiveIP(vmIps string, vmSvc int) (string, error) {
	ips := regexp.MustCompile(ipv4Regex).FindAllString(vmIps, -1)
	for _, ip := range ips {
		vmIp := fmt.Sprintf("%v:%v", ip, vmSvc)
		//fmt.Println("Try connection into:", vmIp)

		conn, err := net.DialTimeout("tcp", vmIp, tcpTimeout)
		if err != nil {
			//fmt.Println("TCP connection failed:", err)
			continue
		}

		defer conn.Close()
		return vmIp, nil
	}

	return "", errors.New("VM service unreachable")
}

func Difference(a, b []string) []string {
	m := make(map[string]bool)
	for _, item := range b {
		m[item] = true
	}

	var diff []string
	for _, item := range a {
		if !m[item] {
			diff = append(diff, item)
		}
	}
	return diff
}

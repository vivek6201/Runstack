//go:build !linux

package main

import (
	"fmt"
	"net"
)

func listenVsock(port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf(":%d", port))
}

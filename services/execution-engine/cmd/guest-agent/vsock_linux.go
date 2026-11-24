//go:build linux

package main

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

func listenVsock(port int) (net.Listener, error) {
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("vsock socket creation failed: %w", err)
	}

	sa := &unix.SockaddrVM{
		CID:  unix.VMADDR_CID_ANY,
		Port: uint32(port),
	}

	if err := unix.Bind(fd, sa); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("vsock bind to port %d failed: %w", port, err)
	}

	if err := unix.Listen(fd, 32); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("vsock listen on port %d failed: %w", port, err)
	}

	return &vsockListener{
		fd:   fd,
		port: port,
	}, nil
}

type vsockListener struct {
	fd   int
	port int
}

func (l *vsockListener) Accept() (net.Conn, error) {
	nfd, sa, err := unix.Accept(l.fd)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(nfd), "vsock-conn")
	var remoteCID, remotePort uint32
	if vmAddr, ok := sa.(*unix.SockaddrVM); ok {
		remoteCID = vmAddr.CID
		remotePort = vmAddr.Port
	}
	return &vsockConn{
		File:       file,
		localPort:  l.port,
		remoteCID:  remoteCID,
		remotePort: remotePort,
	}, nil
}

func (l *vsockListener) Close() error {
	return unix.Close(l.fd)
}

func (l *vsockListener) Addr() net.Addr {
	return &vsockAddr{cid: unix.VMADDR_CID_ANY, port: uint32(l.port)}
}

type vsockConn struct {
	*os.File
	localPort  int
	remoteCID  uint32
	remotePort uint32
}

func (c *vsockConn) LocalAddr() net.Addr {
	return &vsockAddr{cid: unix.VMADDR_CID_ANY, port: uint32(c.localPort)}
}

func (c *vsockConn) RemoteAddr() net.Addr {
	return &vsockAddr{cid: c.remoteCID, port: c.remotePort}
}

type vsockAddr struct {
	cid  uint32
	port uint32
}

func (a *vsockAddr) Network() string {
	return "vsock"
}

func (a *vsockAddr) String() string {
	return fmt.Sprintf("%d:%d", a.cid, a.port)
}

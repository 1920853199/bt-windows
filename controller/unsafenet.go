package controller

import (
        "net"
        "sync"
        "syscall"
        "unsafe"
)

// copied from go/src/pkg/net/fd_windows.go
type ioResult struct {
        qty uint32
        err error
}

// copied from go/src/pkg/net/fd_windows.go
type netFD struct {
        // locking/lifetime of sysfd
        sysmu   sync.Mutex
        sysref  int
        closing bool

        // immutable until Close
        sysfd       syscall.Handle
        family      int
        sotype      int
        isConnected bool
        net         string
        laddr       net.Addr
        raddr       net.Addr
        resultc     [2]chan ioResult
        errnoc      [2]chan error

        // owned by client
        rdeadline int64
        rio       sync.Mutex
        wdeadline int64
        wio       sync.Mutex
}

// copied from go/src/pkg/net/udpsock_posix.go
type UDPConn struct {
    fd *netFD
}

// function to get fd
func GetFD(conn *net.UDPConn) syscall.Handle {
    c := (*UDPConn)(unsafe.Pointer(conn))
    return c.fd.sysfd
}
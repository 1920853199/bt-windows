package controller

import (
        "golang.org/x/sys/windows"
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
type FD struct {
        // Lock sysfd and serialize access to Read and Write methods.
        fdmu fdMutex

        // System file descriptor. Immutable until Close.
        Sysfd syscall.Handle

        // Read operation.
        rop operation
        // Write operation.
        wop operation

        // I/O poller.
        pd pollDesc

        // Used to implement pread/pwrite.
        l sync.Mutex

        // For console I/O.
        lastbits       []byte   // first few bytes of the last incomplete rune in last write
        readuint16     []uint16 // buffer to hold uint16s obtained with ReadConsole
        readbyte       []byte   // buffer to hold decoding of readuint16 from utf16 to utf8
        readbyteOffset int      // readbyte[readOffset:] is yet to be consumed with file.Read

        // Semaphore signaled when file is closed.
        csema uint32

        skipSyncNotif bool

        // Whether this is a streaming descriptor, as opposed to a
        // packet-based descriptor like a UDP socket.
        IsStream bool

        // Whether a zero byte read indicates EOF. This is false for a
        // message based socket connection.
        ZeroReadIsEOF bool

        // Whether this is a file rather than a network socket.
        isFile bool

        // The kind of this file.
        kind fileKind
}

// copied from go/src/pkg/net/fd_windows.go
type netFD struct {
        pfd FD

        // immutable until Close
        family      int
        sotype      int
        isConnected bool // handshake completed or use of association with peer
        net         string
        laddr       net.Addr
        raddr       net.Addr
}
// copied from go/src/pkg/net/udpsock_posix.go
type UDPConn struct {
    fd *netFD
}
type fdMutex struct {
        state uint64
        rsema uint32
        wsema uint32
}
type operation struct {
        // Used by IOCP interface, it must be first field
        // of the struct, as our code rely on it.
        o syscall.Overlapped

        // fields used by runtime.netpoll
        runtimeCtx uintptr
        mode       int32
        errno      int32
        qty        uint32

        // fields used only by net package
        fd     *FD
        errc   chan error
        buf    syscall.WSABuf
        msg    windows.WSAMsg
        sa     syscall.Sockaddr
        rsa    *syscall.RawSockaddrAny
        rsan   int32
        handle syscall.Handle
        flags  uint32
        bufs   []syscall.WSABuf
}
type pollDesc struct {
        runtimeCtx uintptr
}
// fileKind describes the kind of file.
type fileKind byte

// function to get fd
func GetFD(conn *net.UDPConn) syscall.Handle {

    c := (*UDPConn)(unsafe.Pointer(conn))

    return c.fd.pfd.Sysfd
}
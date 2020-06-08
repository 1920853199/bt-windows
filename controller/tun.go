package controller

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

const (
	DefaultMTU = 1420
)

type NativeTun struct {
	fd     int
	name   string
	errors chan error // async error handling
}

func (tun *NativeTun) MTU() (int, error) {
	return DefaultMTU, nil
}

func (tun *NativeTun) Write(d []byte) (int, error) {
	if IsiOS {
		var b = make([]byte, 0, len(d)+4)
		b = append(b, []byte{0, 0, 0, 2}...)
		b = append(b, d...)
		d = b
	}

	return windows.Write(windows.Handle(uintptr(unsafe.Pointer(&tun.fd))), d)
}

func (tun *NativeTun) Read(d []byte) ([]byte, error) {
	offset := 0
	if IsiOS {
		offset = 4
	}

	select {
	case err := <-tun.errors:
		return nil, err
	default:
		n, err := windows.Read(windows.Handle(uintptr(unsafe.Pointer(&tun.fd))), d)
		if err != nil || n == 0 {
			return nil, err
		}

		return d[offset:n], nil
	}
}

func (tun *NativeTun) Close() error {
	return nil
}

func CreateTUN(fd int) (*NativeTun, error) {
	//windows.SetNonblock(fd, false)

	device := &NativeTun{
		fd:     fd,
		errors: make(chan error, 5),
	}

	return device, nil
}

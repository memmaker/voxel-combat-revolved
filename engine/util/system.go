package util

import (
	"fmt"
	"unsafe"
)

type Endianess int

const (
	LittleEndian Endianess = iota
	BigEndian
)

func (e Endianess) ToString() string {
	switch e {
	case LittleEndian:
		return "LittleEndian"
	case BigEndian:
		return "BigEndian"
	}
	return "Unknown"
}
func GetSystemNativeEndianess() (Endianess, error) {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)
	switch buf {
	case [2]byte{0xCD, 0xAB}:
		return LittleEndian, nil
	case [2]byte{0xAB, 0xCD}:
		return BigEndian, nil
	default:
		return 0, fmt.Errorf("unable to determine endianness")
	}
}

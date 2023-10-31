package cni

import "net"

func BytesToUint16(bytes []byte, offset int) uint16 {
	return uint16(bytes[offset])<<8 |
		uint16(bytes[offset+1])
}

func BytesToUint32(bytes []byte, offset int) uint32 {
	return uint32(bytes[offset])<<24 |
		uint32(bytes[offset+1])<<16 |
		uint32(bytes[offset+2])<<8 |
		uint32(bytes[offset+3])
}

func Uint16ToBytes(value uint16) []byte {
	return []byte{
		byte((value & uint16(0xff00)) >> 8),
		byte(value & uint16(0x00ff)),
	}
}

func Uint32ToBytes(value uint32) []byte {
	return []byte{
		byte((value & uint32(0xff000000)) >> 24),
		byte((value & uint32(0x00ff0000)) >> 16),
		byte((value & uint32(0x0000ff00)) >> 8),
		byte(value & uint32(0x000000ff)),
	}
}

func IPToArray4(ip net.IP) [4]byte {
	return [4]byte{
		ip[0],
		ip[1],
		ip[2],
		ip[3],
	}
}

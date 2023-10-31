package cni

import (
	"errors"
	"fmt"
	"net"
)

const (
	minimumHeaderLen = 5
)

type IPFrame struct {
	// 4 bits
	Version byte

	// 4 bits
	// in 4bytes
	HeaderLen byte

	// 8 bits
	Tos byte

	// 16 bits
	// in byte
	TotalLen uint16

	// 16 bits
	Identification uint16

	// 3 bits
	Flag byte

	// 13 bits
	Offset uint16

	// 8 bits
	Ttl byte

	// 8 bits
	Protocol byte

	// 16 bits
	HeaderCheckSum uint16

	// 32 bits
	Source net.IP

	// 32 bits
	Target net.IP

	// options
	Options []Option

	// payload
	Payload []byte
}

type Option struct {
	Key   uint16
	Value uint16
}

// ParseIPFrame TUN - IPv4 Packet:
//
//	+---------------------------------------------------------------------------------------------------------------+
//	|       | Octet |           0           |           1           |           2           |           3           |
//	| Octet |  Bit  |00|01|02|03|04|05|06|07|08|09|10|11|12|13|14|15|16|17|18|19|20|21|22|23|24|25|26|27|28|29|30|31|
//	+---------------------------------------------------------------------------------------------------------------+
//	|   0   |   0   |  Version  |    IHL    |      DSCP       | ECN |                 Total  Length                 |
//	+---------------------------------------------------------------------------------------------------------------+
//	|   4   |  32   |                Identification                 | Flags  |           Fragment Offset            |
//	+---------------------------------------------------------------------------------------------------------------+
//	|   8   |  64   |     Time To Live      |       Protocol        |                Header Checksum                |
//	+---------------------------------------------------------------------------------------------------------------+
//	|  12   |  96   |                                       Source IP Address                                       |
//	+---------------------------------------------------------------------------------------------------------------+
//	|  16   |  128  |                                    Destination IP Address                                     |
//	+---------------------------------------------------------------------------------------------------------------+
//	|  20   |  160  |                                     Options (if IHL > 5)                                      |
//	+---------------------------------------------------------------------------------------------------------------+
//	|  24   |  192  |                                                                                               |
//	|  30   |  224  |                                            Payload                                            |
//	|  ...  |  ...  |                                                                                               |
//	+---------------------------------------------------------------------------------------------------------------+
func ParseIPFrame(buffer ByteBuffer) (*IPFrame, error) {
	// no enough data for first 4 bytes
	if buffer.ReadableBytes() < 4 {
		return nil, nil
	}

	var version byte
	var headerLen byte
	var tos byte
	var totalLen uint16
	var identification uint16
	var flag byte
	var offset uint16
	var ttl byte
	var protocol byte
	var headerCheckSum uint16
	var source net.IP
	var target net.IP
	var options []Option
	var payload []byte

	first4Bytes := make([]byte, 4)

	buffer.Mark()
	buffer.Read(first4Bytes)
	buffer.Recover()

	// 0: version(4) / headerLen(4) / tos(8) / totalLen(16)
	version = (0xf0 & first4Bytes[0]) >> 4
	//if version != 4 {
	//	return nil, errors.New("non ipv4")
	//}
	headerLen = 0x0f & first4Bytes[0]
	if headerLen < minimumHeaderLen {
		return nil, errors.New("header len is small than " + string(rune(minimumHeaderLen)))
	}
	if buffer.ReadableBytes() < int(headerLen*4) {
		return nil, nil
	}
	tos = first4Bytes[1]
	totalLen = BytesToUint16(first4Bytes, 2)
	if buffer.ReadableBytes() < int(totalLen) {
		return nil, nil
	}

	bytes := make([]byte, totalLen)
	buffer.Read(bytes)

	// 1: identification(16) / flag(3) / offset(13)
	identification = BytesToUint16(bytes, 4)
	flag = (bytes[6] & 0xe0) >> 5
	offset |= uint16(bytes[6]&0x1f) << 8
	offset |= uint16(bytes[7] & 0xff)

	// 2: ttl(8) / protocol(8) / headerCheckSum(16)
	ttl = bytes[8]
	protocol = bytes[9]
	headerCheckSum = BytesToUint16(bytes, 10)

	// 3: Source(32)
	source = bytes[12:16]

	// 4: Target(32)
	target = bytes[16:20]

	// 5-headerLen: options(32...)
	if headerLen > minimumHeaderLen {
		options = make([]Option, headerLen-minimumHeaderLen)
		for line := minimumHeaderLen; line < int(headerLen); line += 1 {
			index := line * 4
			options[line-minimumHeaderLen] = Option{
				Key:   BytesToUint16(bytes, index),
				Value: BytesToUint16(bytes, index+2),
			}
		}
	}

	// headerLen-end: payload
	payload = bytes[headerLen*4 : totalLen]

	return &IPFrame{
		Version:        version,
		HeaderLen:      headerLen,
		Tos:            tos,
		TotalLen:       totalLen,
		Identification: identification,
		Flag:           flag,
		Offset:         offset,
		Ttl:            ttl,
		Protocol:       protocol,
		HeaderCheckSum: headerCheckSum,
		Source:         source,
		Target:         target,
		Options:        options,
		Payload:        payload,
	}, nil
}

func (frame *IPFrame) Strings() string {
	return fmt.Sprintf("IPFrame {\n"+
		"\tversion=%d\n"+
		"\theaderLen=%d\n"+
		"\ttos=%d\n"+
		"\ttotalLen=%d\n"+
		"\tidentification=%d\n"+
		"\tflag=%d\n"+
		"\toffset=%d\n"+
		"\tttl=%d\n"+
		"\tprotocol=%d\n"+
		"\theaderCheckSum=%d\n"+
		"\tsource=%s\n"+
		"\ttarget=%s\n"+
		"\tpayloadLen=%d\n"+
		"}\n",
		frame.Version,
		frame.HeaderLen,
		frame.Tos,
		frame.TotalLen,
		frame.Identification,
		frame.Flag,
		frame.Offset,
		frame.Ttl,
		frame.Protocol,
		frame.HeaderCheckSum,
		frame.Source.String(),
		frame.Target.String(),
		len(frame.Payload))
}

func (frame *IPFrame) GetProtocol() string {
	return fmt.Sprintf("\tprotocol=%d\n", frame.Protocol)
}

func (frame *IPFrame) GetSourceIP() string {
	return frame.Source.String()
}

func (frame *IPFrame) GetTargetIP() string {
	return frame.Target.String()
}

func (frame *IPFrame) GetPayloadLen() int {
	return len(frame.Payload)
}

func (frame *IPFrame) ToBytes() []byte {
	bytes := make([]byte, 0)

	// 0: version(4) / headerLen(4) / tos(8) / totalLen(16)
	bytes = append(bytes, (frame.Version<<4)+frame.HeaderLen)
	bytes = append(bytes, frame.Tos)
	bytes = append(bytes, Uint16ToBytes(frame.TotalLen)...)

	// 1: identification(16) / flag(3) / offset(13)
	bytes = append(bytes, Uint16ToBytes(frame.Identification)...)
	bytes = append(bytes, (frame.Flag<<5)|byte((frame.Offset&uint16(0x1f00))>>8))
	bytes = append(bytes, byte(frame.Offset&0xff))

	// 2: ttl(8) / protocol(8) / headerCheckSum(16)
	bytes = append(bytes, frame.Ttl)
	bytes = append(bytes, frame.Protocol)
	bytes = append(bytes, 0x00, 0x00)

	// 3: Source(32)
	bytes = append(bytes, frame.Source[:]...)

	// 4: Target(32)
	bytes = append(bytes, frame.Target[:]...)

	// options
	if frame.Options != nil {
		bytes = append(bytes, toBytes(frame.Options)...)
	}

	checkSum := toCheckSum(bytes)
	copy(bytes[10:12], Uint16ToBytes(checkSum))
	frame.HeaderCheckSum = checkSum

	// payload
	bytes = append(bytes, frame.Payload...)

	return bytes
}

func toBytes(options []Option) []byte {
	if options == nil {
		return make([]byte, 0)
	}

	bytes := make([]byte, 0)

	for i := 0; i < len(options); i += 1 {
		bytes = append(bytes, options[i].toBytes()...)
	}

	return bytes
}

func (option Option) toBytes() []byte {
	bytes := make([]byte, 0)

	bytes = append(bytes, Uint16ToBytes(option.Key)...)
	bytes = append(bytes, Uint16ToBytes(option.Value)...)

	return bytes
}

func toCheckSum(bytes []byte) uint16 {
	var checkSum uint32 = 0

	num := len(bytes) >> 1

	for i := 0; i < num; i += 1 {
		checkSum += uint32(BytesToUint16(bytes, i*2))
	}

	return ^(uint16(checkSum>>16) + uint16(checkSum&0xffff))
}

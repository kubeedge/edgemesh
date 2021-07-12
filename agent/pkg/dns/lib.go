// TODO: use https://github.com/miekg/dns to replace
package dns

import (
	"encoding/binary"
	"errors"
	"net"
	"unsafe"
)

type Event int

var (
	// QR: 0 represents query, 1 represents response
	dnsQR       = uint16(0x8000)
	oneByteSize = uint16(1)
	twoByteSize = uint16(2)
	ttl         = uint32(64)
)

const (
	// 1 for ipv4
	aRecord           = 1
	bufSize           = 1024
	errNotImplemented = uint16(0x0004)
	errRefused        = uint16(0x0005)
	eventNothing      = Event(0)
	eventUpstream     = Event(1)
	eventNxDomain     = Event(2)
)

type dnsHeader struct {
	id      uint16
	flags   uint16
	qdCount uint16
	anCount uint16
	nsCount uint16
	arCount uint16
}

// isAQuery judges if the dns pkg is a query
func (h *dnsHeader) isAQuery() bool {
	return h.flags&dnsQR != dnsQR
}

// getHeader gets dns pkg head
func (h *dnsHeader) getHeader(req []byte) {
	h.id = binary.BigEndian.Uint16(req[0:2])
	h.flags = binary.BigEndian.Uint16(req[2:4])
	h.qdCount = binary.BigEndian.Uint16(req[4:6])
	h.anCount = binary.BigEndian.Uint16(req[6:8])
	h.nsCount = binary.BigEndian.Uint16(req[8:10])
	h.arCount = binary.BigEndian.Uint16(req[10:12])
}

// convertQueryRsp converts a dns question head to a response head
func (h *dnsHeader) convertQueryRsp(isRsp bool) {
	if isRsp {
		h.flags |= dnsQR
	} else {
		h.flags |= dnsQR
	}
}

// setAnswerNum sets the answer num for dns head
func (h *dnsHeader) setAnswerNum(num uint16) {
	h.anCount = num
}

// setRspRCode sets dns response return code
func (h *dnsHeader) setRspRCode(que *dnsQuestion) {
	if que.qType != aRecord {
		h.flags &= ^errNotImplemented
		h.flags |= errNotImplemented
	} else if que.event == eventNxDomain {
		h.flags &= ^errRefused
		h.flags |= errRefused
	}
}

// getByteFromDNSHeader converts dnsHeader to bytes
func (h *dnsHeader) getByteFromDNSHeader() (rspHead []byte) {
	rspHead = make([]byte, unsafe.Sizeof(*h))

	idxTransactionID := unsafe.Sizeof(h.id)
	idxFlags := unsafe.Sizeof(h.flags) + idxTransactionID
	idxQDCount := unsafe.Sizeof(h.anCount) + idxFlags
	idxANCount := unsafe.Sizeof(h.anCount) + idxQDCount
	idxNSCount := unsafe.Sizeof(h.nsCount) + idxANCount
	idxARCount := unsafe.Sizeof(h.arCount) + idxNSCount

	binary.BigEndian.PutUint16(rspHead[:idxTransactionID], h.id)
	binary.BigEndian.PutUint16(rspHead[idxTransactionID:idxFlags], h.flags)
	binary.BigEndian.PutUint16(rspHead[idxFlags:idxQDCount], h.qdCount)
	binary.BigEndian.PutUint16(rspHead[idxQDCount:idxANCount], h.anCount)
	binary.BigEndian.PutUint16(rspHead[idxANCount:idxNSCount], h.nsCount)
	binary.BigEndian.PutUint16(rspHead[idxNSCount:idxARCount], h.arCount)
	return
}

type dnsQuestion struct {
	from    *net.UDPAddr
	head    *dnsHeader
	name    []byte
	queByte []byte
	qType   uint16
	qClass  uint16
	event   Event
}

// getQuestion gets a dns question
func (q *dnsQuestion) getQuestion(req []byte, offset uint16, head *dnsHeader) {
	ost := offset
	tmp := ost
	ost = q.getQName(req, ost)
	q.qType = binary.BigEndian.Uint16(req[ost : ost+twoByteSize])
	ost += twoByteSize
	q.qClass = binary.BigEndian.Uint16(req[ost : ost+twoByteSize])
	ost += twoByteSize
	q.head = head
	q.queByte = req[tmp:ost]
}

// getQName gets dns question qName
func (q *dnsQuestion) getQName(req []byte, offset uint16) uint16 {
	ost := offset

	for {
		// one byte to suggest length
		qbyte := uint16(req[ost])

		// qName ends with 0x00, and 0x00 should not be included
		if qbyte == 0x00 {
			q.name = q.name[:uint16(len(q.name))-oneByteSize]
			return ost + oneByteSize
		}
		// step forward one more byte and get the real stuff
		ost += oneByteSize
		q.name = append(q.name, req[ost:ost+qbyte]...)
		// add "." symbol
		q.name = append(q.name, 0x2e)
		ost += qbyte
	}
}

// modifyRspPrefix generates a dns response head
func modifyRspPrefix(que *dnsQuestion) (pre []byte) {
	if que == nil {
		return
	}
	// use head in que
	rspHead := que.head
	rspHead.convertQueryRsp(true)
	if que.qType == aRecord {
		rspHead.setAnswerNum(1)
	} else {
		rspHead.setAnswerNum(0)
	}

	rspHead.setRspRCode(que)
	pre = rspHead.getByteFromDNSHeader()

	pre = append(pre, que.queByte...)
	return
}

// parseDNSQuery converts bytes to *dnsQuestion
func parseDNSQuery(req []byte) (que *dnsQuestion, err error) {
	head := &dnsHeader{}
	head.getHeader(req)
	if !head.isAQuery() {
		return nil, errors.New("not a dns query, ignore")
	}
	que = &dnsQuestion{
		event: eventNothing,
	}
	// Generally, when the recursive DNS server requests upward, it may
	// initiate a resolution request for multiple aliases/domain names
	// at once, Edge DNS does not need to process a message that carries
	// multiple questions at a time.
	if head.qdCount != 1 {
		que.event = eventUpstream
		return
	}

	offset := uint16(unsafe.Sizeof(dnsHeader{}))
	// DNS NS <ROOT> operation
	if req[offset] == 0x0 {
		que.event = eventUpstream
		return
	}
	que.getQuestion(req, offset, head)
	err = nil
	return
}

type dnsAnswer struct {
	name    []byte
	qType   uint16
	qClass  uint16
	ttl     uint32
	dataLen uint16
	addr    []byte
}

// getAnswer generates answer for the dns question
func (da *dnsAnswer) getAnswer() (answer []byte) {
	answer = make([]byte, 0)

	if da.qType == aRecord {
		answer = append(answer, 0xc0)
		answer = append(answer, 0x0c)

		tmp16 := make([]byte, 2)
		tmp32 := make([]byte, 4)

		binary.BigEndian.PutUint16(tmp16, da.qType)
		answer = append(answer, tmp16...)
		binary.BigEndian.PutUint16(tmp16, da.qClass)
		answer = append(answer, tmp16...)
		binary.BigEndian.PutUint32(tmp32, da.ttl)
		answer = append(answer, tmp32...)
		binary.BigEndian.PutUint16(tmp16, da.dataLen)
		answer = append(answer, tmp16...)
		answer = append(answer, da.addr...)
	}

	return answer
}

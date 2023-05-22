package demo

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

const headLen = 3
const headPackageTypeIdx = 0

type ConnReader struct {
	conn    net.Conn
	headBuf []byte
}

func NewConnReader(conn net.Conn) *ConnReader {
	return &ConnReader{
		conn:    conn,
		headBuf: make([]byte, headLen),
	}
}

func (r *ConnReader) Read() (p Package, err error) {
	_, err = io.ReadFull(r.conn, r.headBuf)
	if err != nil {
		return
	}
	bodyLen := binary.BigEndian.Uint16(r.headBuf[headPackageTypeIdx+1:])
	p.All = make([]byte, headLen+int(bodyLen))
	copy(p.All[:headLen], r.headBuf)
	_, err = io.ReadFull(r.conn, p.All[headLen:])

	p.Type = r.headBuf[headPackageTypeIdx]
	p.Body = p.All[headLen:]

	return
}

type Package struct {
	All  []byte
	Type PackageType
	Body []byte
}

func PackHeartbeatPackage() []byte {
	return PackPackage(Heartbeat, nil)
}

func PackPackage(t PackageType, body []byte) []byte {
	bs := make([]byte, headLen+len(body))
	binary.BigEndian.PutUint16(bs[headPackageTypeIdx+1:headLen], uint16(len(body)))
	bs[headPackageTypeIdx] = t
	copy(bs[headLen:], body)
	return bs
}

func NewPackage(t PackageType, body []byte) Package {
	pkg := Package{}
	pkg.All = PackPackage(t, body)
	pkg.Type = t
	pkg.Body = body
	return pkg
}

func PackageAttachSessionId(p *Package, sessionId int64) {
	sessionIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sessionIdBytes, uint64(sessionId))
	p.All = append(p.All, sessionIdBytes...)
	binary.BigEndian.PutUint16(p.All[headPackageTypeIdx+1:headLen], uint16(len(p.Body)+8))
	p.Body = p.All[headLen:]
}

func PackagePickSessionId(p *Package) (sessionId int64) {
	sessionId = int64(binary.BigEndian.Uint64(p.All[len(p.All)-8:]))
	binary.BigEndian.PutUint16(p.All[headPackageTypeIdx+1:headLen], uint16(len(p.Body)-8))
	p.All = p.All[:len(p.All)-8]
	p.Body = p.All[headLen:]
	return
}

type SimpleMsg struct {
	Cmd Cmd
	Msg string
}

func DecodeSimpleMsg(body []byte) (result SimpleMsg, err error) {
	arr := strings.Split(string(body), ":")
	if len(arr) != 2 {
		err = fmt.Errorf("输入内容格式错误")
		return
	}
	i, err := strconv.Atoi(arr[0])
	if err != nil {
		err = fmt.Errorf("输入内容格式错误")
		return
	}
	result.Cmd = Cmd(i)
	result.Msg = arr[1]
	return
}

func EncodeSimpleMsg(in SimpleMsg) []byte {
	return []byte(strconv.Itoa(int(in.Cmd)) + ":" + in.Msg)
}

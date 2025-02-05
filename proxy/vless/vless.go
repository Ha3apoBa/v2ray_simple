package vless

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"

	"github.com/hahahrfool/v2ray_simple/common"
	"github.com/hahahrfool/v2ray_simple/proxy"
)

const Name = "vless"

const (
	Cmd_CRUMFURS byte = 4 // start from vless v1

	CRUMFURS_ESTABLISHED byte = 20

	CRUMFURS_Established_Str = "CRUMFURS_Established"
)

type UserConn struct {
	net.Conn
	uuid         [16]byte
	convertedStr string
	version      int
	isUDP        bool
	isServerEnd  bool //for v0

	// udpUnreadPart 不为空，则表示上一次读取没读完整个包（给Read传入的buf太小），接着读
	udpUnreadPart []byte //for udp

	bufr            *bufio.Reader //for udp
	isntFirstPacket bool          //for v0

	hasAdvancedLayer bool //for v1, 在用ws或grpc时，这个开关保持打开
}

func (uc *UserConn) GetProtocolVersion() int {
	return uc.version
}
func (uc *UserConn) GetIdentityStr() string {
	if uc.convertedStr == "" {
		uc.convertedStr = proxy.UUIDToStr(uc.uuid)
	}

	return uc.convertedStr
}

//如果是udp，则是多线程不安全的，如果是tcp，则安不安全看底层的链接。
// 这里规定，如果是UDP，则 每Write一遍，都要Write一个 完整的UDP 数据包
func (uc *UserConn) Write(p []byte) (int, error) {

	if uc.version == 0 {

		originalSupposedWrittenLenth := len(p)

		var writeBuf *bytes.Buffer

		if uc.isServerEnd && !uc.isntFirstPacket {
			uc.isntFirstPacket = true

			writeBuf = common.GetBuf()

			//v0 中，服务端的回复的第一个包也是要有数据头的(和客户端的handshake类似，只是第一个包有)，第一字节版本，第二字节addon长度（都是0）

			writeBuf.WriteByte(0)
			writeBuf.WriteByte(0)

		}

		if !uc.isUDP {

			if writeBuf != nil {
				writeBuf.Write(p)

				_, err := uc.Conn.Write(writeBuf.Bytes()) //“直接return这个的长度” 是错的，因为写入长度只能小于等于len(p)

				common.PutBuf(writeBuf)

				if err != nil {
					return 0, err
				}
				return originalSupposedWrittenLenth, nil

			} else {
				_, err := uc.Conn.Write(p) //“直接return这个的长度” 是错的，因为写入长度只能小于等于len(p)

				if err != nil {
					return 0, err
				}
				return originalSupposedWrittenLenth, nil
			}

		} else {
			l := int16(len(p))
			if writeBuf == nil {
				writeBuf = common.GetBuf()
			}

			writeBuf.WriteByte(byte(l >> 8))
			writeBuf.WriteByte(byte(l << 8 >> 8))
			writeBuf.Write(p)

			_, err := uc.Conn.Write(writeBuf.Bytes()) //“直接return这个的长度” 是错的，因为写入长度只能小于等于len(p)

			common.PutBuf(writeBuf)

			if err != nil {
				return 0, err
			}
			return originalSupposedWrittenLenth, nil
		}

	} else {
		if uc.isUDP && !uc.hasAdvancedLayer {

			// 这里暂时认为包裹它的连接是 tcp或者tls，而不是udp，如果udp的话，就不需要考虑粘包问题了，比如socks5的实现
			// 我们目前认为只有tls是最防墙的，而且 魔改tls是有毒的，所以反推过来，这里udp就必须加长度头。

			// 目前是这个样子。之后verysimple实现了websocket和grpc后，会添加判断，如果连接是websocket或者grpc连接，则不再加长度头

			// tls和tcp都是基于流的，可以分开写两次，不需要buf存在；如果连接是websocket或者grpc的话，直接传输。

			l := int16(len(p))
			var lenBytes []byte

			if l <= 255 {
				lenBytes = []byte{0, byte(l)}
			}

			lenBytes = []byte{byte(l >> 8), byte(l << 8 >> 8)}

			_, err := uc.Conn.Write(lenBytes)
			if err != nil {
				return 0, err
			}

			return uc.Conn.Write(p)

		}
		return uc.Conn.Write(p)

	}
}

//如果是udp，则是多线程不安全的，如果是tcp，则安不安全看底层的链接。
// 这里规定，如果是UDP，则 每次 Read 得到的都是一个 完整的UDP 数据包，除非p给的太小……
func (uc *UserConn) Read(p []byte) (int, error) {

	if uc.version == 0 {

		if !uc.isUDP {

			if !uc.isServerEnd && !uc.isntFirstPacket {

				uc.isntFirstPacket = true

				bs := common.GetPacket()
				n, e := uc.Conn.Read(bs)

				if e != nil {
					return 0, e
				}

				if n < 2 {
					return 0, errors.New("vless response head too short")
				}
				n = copy(p, bs[2:n])
				common.PutPacket(bs)
				return n, nil

			}

			return uc.Conn.Read(p)
		} else {

			if uc.bufr == nil {
				uc.bufr = bufio.NewReader(uc.Conn)
			}

			if len(uc.udpUnreadPart) > 0 {
				copiedN := copy(p, uc.udpUnreadPart)
				if copiedN < len(uc.udpUnreadPart) {
					uc.udpUnreadPart = uc.udpUnreadPart[copiedN:]
				} else {
					uc.udpUnreadPart = nil
				}
				return copiedN, nil
			}

			//v0 先读取vless响应头，再读取udp长度头

			if !uc.isServerEnd && !uc.isntFirstPacket {
				uc.isntFirstPacket = true

				_, err := uc.bufr.ReadByte() //version byte
				if err != nil {
					return 0, err
				}
				_, err = uc.bufr.ReadByte() //addon len byte
				if err != nil {
					return 0, err
				}

			}

			return uc.readudp_withLenthHead(p)
		}

	} else {
		if uc.isUDP && !uc.hasAdvancedLayer {

			if len(uc.udpUnreadPart) > 0 {
				copiedN := copy(p, uc.udpUnreadPart)
				if copiedN < len(uc.udpUnreadPart) {
					uc.udpUnreadPart = uc.udpUnreadPart[copiedN:]
				} else {
					uc.udpUnreadPart = nil
				}
				return copiedN, nil
			}

			return uc.readudp_withLenthHead(p)
		}
		return uc.Conn.Read(p)

	}
}

func (uc *UserConn) readudp_withLenthHead(p []byte) (int, error) {
	if uc.bufr == nil {
		uc.bufr = bufio.NewReader(uc.Conn)
	}
	b1, err := uc.bufr.ReadByte()
	if err != nil {
		return 0, err
	}
	b2, err := uc.bufr.ReadByte()
	if err != nil {
		return 0, err
	}

	l := int(int16(b1)<<8 + int16(b2))
	bs := common.GetBytes(l)
	n, err := io.ReadFull(uc.bufr, bs)
	if err != nil {
		return 0, err
	}
	/*// 测试代码
	if uc.version == 1 {
		log.Println("read", bs[:n], string(bs[:n]))

	}*/

	copiedN := copy(p, bs)
	if copiedN < n { //p is short
		uc.udpUnreadPart = bs[copiedN:]
	}

	return copiedN, nil
}

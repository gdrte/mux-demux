package muxdemux

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"net"
	"strconv"
	"strings"
)

/*
Mux/Demux is based on a simple protocol
Message Structure
~~~~~~~~~~~~~~~~~
MSG
<<streamname>>
<<id>>
<<length>>
<<message>>
END

Stream structure
~~~~~~~~~~~~~~~~
STREAMS
+<<stream-1>>
-<<stream-2>>
END
*/

const (
	NEW_LINE = '\n'
)

type MuxDemuxStream struct {
	In   chan Message
	Out  chan Message
	exit chan bool
	Name string
}

type MuxDemux struct {
	conn    net.Conn
	in      chan Message
	out     chan Message
	streams map[string]*MuxDemuxStream
}

type Message struct {
	SName  string
	Id     string
	Length int
	Body   []byte
}

func New(conn net.Conn) *MuxDemux {
	return &MuxDemux{conn: conn, streams: make(map[string]*MuxDemuxStream), in: make(chan Message, 10), out: make(chan Message, 10)}
}

func (mxdx *MuxDemux) Close() {
	mxdx.conn.Close()
	for key, _ := range mxdx.streams {
		delete(mxdx.streams, key)
	}
}

func (mxdx *MuxDemux) NewStream(name string) *MuxDemuxStream {
	return &MuxDemuxStream{Name: name, In: make(chan Message, 1), Out: make(chan Message, 1)}
}

/*
The moment the stream is closed, all the message for this stream will be discarded
*/
func (mxdx *MuxDemux) CloseStream(name string) {
	close(mxdx.streams[name].In)
	close(mxdx.streams[name].Out)
	delete(mxdx.streams, name)
}

func (mxdxs *MuxDemuxStream) write(msg *Message) string {
	msg.SName = mxdxs.Name
	buffer := new(bytes.Buffer)
	buffer.WriteString(fmt.Sprintf("MSG\n%s\n%s\n%d\n", msg.SName, msg.Id, msg.Length))
	buffer.Write(msg.Body)
	buffer.WriteString("END")
	return buffer.String()
}

func (mxdx *MuxDemux) fanOut() {
	for message := range mxdx.in {
		mxdx.streams[message.SName].In <- message
	}
}

func (mxdx *MuxDemux) fanIn() {
	for key, _ := range mxdx.streams {
		msg := <-mxdx.streams[key].Out
		if len(msg.Body) != 0 {
			mxdx.out <- msg
		} else {
			fmt.Printf("Dropped message from :[%s : %s]\n", key, msg.Id)
		}
	}
}

func (mxdx *MuxDemux) pipe() {
	reader := bufio.NewReader(mxdx.conn)

	for {
		if header, err := reader.ReadBytes(NEW_LINE); err == nil {
			msg := Message{}
			switch string(header) {
			case "MSG":
				if sname, err := reader.ReadBytes(NEW_LINE); err == nil {
					msg.SName = string(sname)
				}

				if id, err := reader.ReadBytes(NEW_LINE); err == nil {
					msg.Id = string(id)
				}

				if length, err := reader.ReadBytes(NEW_LINE); err == nil {
					msg.Length, _ = strconv.Atoi(string(length))
				}
				if body, err := reader.ReadBytes(NEW_LINE); err == nil {
					msg.Body = body
				}
				mxdx.in <- msg
				break
			case "STREAMS":
				if buffer, err := reader.ReadBytes(NEW_LINE); err == nil {
					name := string(buffer)
					if strings.HasPrefix(name, "+") {
						mxdx.streams[name] = mxdx.NewStream(name)
					} else if strings.HasPrefix(name, "-") {
						delete(mxdx.streams, name)
					}
				}

				break
			}
		}
	}
}

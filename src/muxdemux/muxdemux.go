package muxdemux

import (
	"bufio"
	"bytes"
	"fmt"
	// "github.com/nu7hatch/gouuid"
	"encoding/base64"
	"net"
	"strconv"
	"strings"
)

// Mux/Demux is based on a simple protocol
// Message Structure
// ~~~~~~~~~~~~~~~~~
// MSG
// <<streamname>>
// <<id>>
// <<length>>
// <<message>>

// Stream structure
// ~~~~~~~~~~~~~~~~
// STREAM
// +<<stream-1>>
// -<<stream-2>>

const (
	NEW_LINE = '\n'
)

type MuxDemuxChannel struct {
	In   chan *Message
	Out  chan *Message
	Exit chan bool
	Err  chan error
	Name string
}

type MuxDemux struct {
	conn    net.Conn
	in      chan *Message
	out     chan *Message
	err     chan error
	exit    chan bool
	streams map[string]*MuxDemuxChannel
}

type Message struct {
	SName  string
	Id     string
	Length int
	Body   []byte
}

func New(conn net.Conn) *MuxDemux {
	mxdx := &MuxDemux{conn: conn, streams: make(map[string]*MuxDemuxChannel), in: make(chan *Message, 5), out: make(chan *Message, 5), exit: make(chan bool, 1)}
	go mxdx.read()
	go mxdx.fanOut()
	go mxdx.write()
	return mxdx
}

func (mxdx *MuxDemux) Close() {
	mxdx.exit <- true
	mxdx.conn.Close()
	for key, _ := range mxdx.streams {
		delete(mxdx.streams, key)
	}
	mxdx.exit <- true //for write
	mxdx.exit <- true //for fanOut
}

func (mxdx *MuxDemux) NewChannel(name string) *MuxDemuxChannel {
	mxdxs := &MuxDemuxChannel{Name: name, In: make(chan *Message, 1), Out: make(chan *Message, 1), Exit: make(chan bool, 1)}
	go mxdx.fanIn(mxdxs)
	mxdx.streams[name] = mxdxs
	return mxdxs
}

func (mxdx *MuxDemux) GetChannel(name string) *MuxDemuxChannel {
	if mxdxs, ok := mxdx.streams[name]; ok {
		return mxdxs
	}
	return nil
}

//The moment the stream is closed, all the message for this stream will be discarded
func (mxdx *MuxDemux) CloseChannel(name string) {
	close(mxdx.streams[name].In)
	close(mxdx.streams[name].Out)
	delete(mxdx.streams, name)
}

func (mxdxs *MuxDemuxChannel) Send(msg *Message) {
	msg.SName = mxdxs.Name
	mxdxs.Out <- msg
	fmt.Println("Message sent..")
}

func (mxdxs *MuxDemuxChannel) Update(conn net.Conn, aORr string) {
	fmt.Fprintf(conn, "STREAM\n%s%s\n", aORr, mxdxs.Name)
	// fmt.Printf("Updated STREAM\n%s%s\n", aORr, mxdxs.Name)
}

func (msg *Message) String() string {
	buffer := new(bytes.Buffer)
	msg.Length = len(msg.Body)
	buffer.WriteString(fmt.Sprintf("MSG\n%s\n%s\n%d\n", msg.SName, msg.Id, msg.Length))
	buffer.WriteString(base64.StdEncoding.EncodeToString(msg.Body))
	buffer.WriteByte(NEW_LINE)
	return buffer.String()
}

func (mxdx *MuxDemux) fanOut() {
	fmt.Printf("Fan OUT started\n")
	for {
		select {
		case message := <-mxdx.in:
			fmt.Printf("Incoming from %s", message.SName)
			if mxdxs, ok := mxdx.streams[message.SName]; ok {
				mxdxs.In <- message
			}
		case <-mxdx.exit:
			fmt.Printf("Closing fanOut routine %v", mxdx)
			return
		}
	}
}

func (mxdx *MuxDemux) fanIn(mxdxs *MuxDemuxChannel) {
	for {
		select {
		case message := <-mxdxs.Out:
			mxdx.out <- message
			break
		case <-mxdx.exit:
			fmt.Println("Closing the channel fanIn")
			return
		case <-mxdxs.Exit:
			fmt.Println("Closing the channel fanIn")
			return
		}
	}
}

func (mxdx *MuxDemux) write() {
	for {
		select {
		case message := <-mxdx.out:
			// fmt.Printf("Sending message..write\n%s", message.String())
			if n, err := fmt.Fprintf(mxdx.conn, message.String()); err != nil {
				fmt.Printf("Bytes written %d, Error is %v", n, err)
			}
		case <-mxdx.exit:
			fmt.Printf("Closing the write for the connection %v", mxdx)
			return
		}
	}
}

func readLine(reader *bufio.Reader) (string, error) {
	if buff, err := reader.ReadBytes(NEW_LINE); err == nil {
		val := string(buff)
		return val[0 : len(string(val))-1], err
	}

	return "", nil
}

func (mxdx *MuxDemux) read() {
	reader := bufio.NewReader(mxdx.conn)
	fmt.Println("Reading incoming..." + mxdx.conn.RemoteAddr().String())
	for {
		if header, err := reader.ReadBytes(NEW_LINE); err == nil {

			switch strings.TrimSpace(string(header)) {
			case "MSG":
				msg := new(Message)
				msg.SName, err = readLine(reader)
				msg.Id, err = readLine(reader)
				length, _ := readLine(reader)
				msg.Length, _ = strconv.Atoi(strings.TrimSpace(string(length)))
				body, _ := readLine(reader)

				if msg.Body, err = base64.StdEncoding.DecodeString(body); err != nil {
					fmt.Printf("Error! Body corrupted %v", err)
				}
				fmt.Println(string(msg.Body))
				mxdx.in <- msg

				break
			case "STREAM":
				if buffer, err := reader.ReadBytes(NEW_LINE); err == nil {
					name := strings.TrimSpace(string(buffer))

					if strings.HasPrefix(name, "+") {
						name = name[1:len(name)]
						if _, ok := mxdx.streams[name]; !ok {
							mxdx.streams[name] = mxdx.NewChannel(name)
							// fmt.Println("New stream " + name)
						}
					} else if strings.HasPrefix(name, "-") {
						delete(mxdx.streams, name)
					}
				} else {
					fmt.Printf("Error :%v", err)
				}
				// fmt.Printf("Len of streams :%d", len(mxdx.streams))

				break
			} //end of switch
		} else {
			if nerr, ok := err.(net.Error); ok {
				fmt.Printf("Error in header :%v", nerr.Temporary())
				return
			}
		}
	}
	fmt.Printf("Quitting read()")
}

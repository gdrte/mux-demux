package muxdemux

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	// "github.com/nu7hatch/gouuid"
	"encoding/base64"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
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

var NotAvailable = errors.New("Channel not available")

type MuxDemuxChannel struct {
	In  chan *Message
	Out chan *Message
	// Exit chan bool
	Err  chan error
	Name string
}

type MuxDemux struct {
	conn net.Conn
	in   chan *Message
	out  chan *Message
	err  chan error
	// exit    chan bool
	streams map[string]*MuxDemuxChannel
	wg      sync.WaitGroup
}

type Message struct {
	SName  string
	Id     string
	Length int
	Body   []byte
}

func New(conn net.Conn) *MuxDemux {
	mxdx := &MuxDemux{conn: conn, streams: make(map[string]*MuxDemuxChannel), in: make(chan *Message, 100), out: make(chan *Message, 100),
		wg: sync.WaitGroup{}}
	go mxdx.read()
	go mxdx.fanOut()
	go mxdx.write()
	mxdx.wg.Add(2)
	time.Sleep(1 * time.Second)
	return mxdx
}

//Closing the MuxDemux will close the underlying tcp connection and opened channels
func (mxdx *MuxDemux) Close() {
	// close(mxdx.exit)
	go func() {
		close(mxdx.in)
		close(mxdx.out)
		for key, _ := range mxdx.streams {
			mxdx.CloseChannel(key)
		}

	}()
	mxdx.conn.Close()
}

func (mxdx *MuxDemux) NewChannel(name string) *MuxDemuxChannel {
	mxdxs := &MuxDemuxChannel{Name: name, In: make(chan *Message, 20), Out: make(chan *Message, 20)}
	go mxdx.fanIn(mxdxs)
	mxdx.streams[name] = mxdxs
	return mxdxs
}

func (mxdx *MuxDemux) GetChannel(name string) (*MuxDemuxChannel, error) {
	if mxdxs, ok := mxdx.streams[name]; ok {
		return mxdxs, nil
	}
	return nil, NotAvailable
}

//The moment the stream is closed, all the message for this stream will be discarded
func (mxdx *MuxDemux) CloseChannel(name string) {
	if mxdxs, ok := mxdx.streams[name]; ok {
		delete(mxdx.streams, name)
		mxdxs.Update(mxdx.conn, "-")

		go func() {
			time.Sleep(1 * time.Second)
			close(mxdxs.In)
			close(mxdxs.Out)
			fmt.Printf("Closed the channel %s and removed the reference\n", name)
		}()
	}
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
	for message := range mxdx.in {
		fmt.Printf("Incoming from %s\n", message.SName)
		if mxdxs, ok := mxdx.streams[message.SName]; ok {
			mxdxs.In <- message
		}

	}
	fmt.Printf("Exiting fanOut \n")
}

func (mxdx *MuxDemux) fanIn(mxdxs *MuxDemuxChannel) {
	for message := range mxdxs.Out {
		mxdx.out <- message
	}
	fmt.Printf("Exiting fanIN \n")
}

func (mxdx *MuxDemux) write() {
OutLoop:
	for message := range mxdx.out {
		// fmt.Printf("Sending message..write\n%s", message.String())
		if n, err := fmt.Fprintf(mxdx.conn, message.String()); err != nil {
			fmt.Printf("Bytes written %d, Error is %v\n", n, err)
			if nerr, ok := err.(*net.OpError); ok {
				if !nerr.Temporary() {
					break OutLoop
				}
			}
		}
	}
	mxdx.wg.Done()
	fmt.Printf("Exiting write \n")
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
Infinity:
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
				if name, err := readLine(reader); err == nil {

					if strings.HasPrefix(name, "+") {
						name = name[1:len(name)]
						if _, ok := mxdx.streams[name]; !ok {
							mxdx.streams[name] = mxdx.NewChannel(name)
							// fmt.Println("New stream " + name)
						}
					} else if strings.HasPrefix(name, "-") {
						name = name[1:len(name)]
						fmt.Println("Remote connection closed this channel")
						mxdx.CloseChannel(name)
					}
				} else {
					fmt.Printf("Error :%v", err)
				}
				// fmt.Printf("Len of streams :%d", len(mxdx.streams))

				break
			} //end of switch
		} else {
			switch err {
			case io.EOF:
				fmt.Printf("Closing tcp connection ..\n")
				mxdx.Close()
				break Infinity
			case io.ErrClosedPipe:
				fmt.Printf("Err closed connection. closing tcp connection ..\n")
				mxdx.Close()
				break Infinity
			default:
				fmt.Printf("Default err:%v\n", err)
				break Infinity
			}

		}
	}
	mxdx.wg.Done()
	fmt.Println("Quitting read()")
}

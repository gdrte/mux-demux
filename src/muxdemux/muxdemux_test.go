package muxdemux

import (
	"fmt"
	uuid "github.com/nu7hatch/gouuid"
	. "gopkg.in/check.v1"
	"net"
	"testing"
	"time"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MuxDemuxSuite struct{}

var _ = Suite(&MuxDemuxSuite{})

func (s *MuxDemuxSuite) SetUpSuite(c *C) {

}

func (s *MuxDemuxSuite) TestMuxDemuxServer(c *C) {
	fmt.Printf("Starting Server..")
	if ln, err := net.Listen("tcp", "localhost:8081"); err == nil {
		for {
			if conn, err := ln.Accept(); err == nil {
				fmt.Printf("Connection accepted..")
				mxdx := New(conn)
				mxdxs := mxdx.NewChannel("Sample")
				mxdxs.Update(conn, "+")
				go func() {
					for {
						msg := new(Message)
						id, _ := uuid.NewV4()
						msg.Id = id.String()
						msg.Body = []byte("Hello how are you?")
						mxdxs.Send(msg)
						time.Sleep(1 * time.Second)
					}
				}()

				go func() {
					for message := range mxdxs.In {
						fmt.Printf("Message:%v", message.String())
					}
				}()

				mxdxs2 := mxdx.NewChannel("Sample2")
				mxdxs2.Update(conn, "+")

				go func() {
					for {
						msg := new(Message)
						id, _ := uuid.NewV4()
						msg.Id = id.String()
						msg.Body = []byte("Hey Deva!!")
						mxdxs2.Send(msg)
						time.Sleep(1 * time.Second)
					}
				}()

				go func() {
					for message := range mxdxs2.In {
						fmt.Printf("Message:%v", message.String())
					}
				}()

				mxdxs3 := mxdx.NewChannel("Sample3")
				mxdxs3.Update(conn, "+")

				go func() {
					for {
						msg := new(Message)
						id, _ := uuid.NewV4()
						msg.Id = id.String()
						msg.Body = []byte("I am from Anantapuram!!")
						mxdxs3.Send(msg)
						time.Sleep(1 * time.Second)
					}
				}()

				go func() {
					for message := range mxdxs3.In {
						fmt.Printf("Message:%v", message.String())
					}
				}()

			} else {
				fmt.Printf("%v", err)
			}
		}
	} else {
		fmt.Printf("%v", err)
	}
}

func (s *MuxDemuxSuite) TestMuxDemuxClient(c *C) {
	fmt.Printf("Starting Client...")
	if conn, err := net.Dial("tcp", "localhost:8081"); err == nil {
		fmt.Printf("Client connection accepted...")
		mxdx := New(conn)
		time.Sleep(1 * time.Second)
		go func() {
			mxdxs := mxdx.GetChannel("Sample")
			for message := range mxdxs.In {
				fmt.Printf("Message:%v", message.String())
				msg := new(Message)
				id, _ := uuid.NewV4()
				msg.Id = id.String()
				msg.Body = []byte("I am fine, Howdy!!")
				mxdxs.Send(msg)
			}
		}()

		go func() {
			mxdxs := mxdx.GetChannel("Sample2")
			for message := range mxdxs.In {
				fmt.Printf("Message:%v", message.String())
				msg := new(Message)
				id, _ := uuid.NewV4()
				msg.Id = id.String()
				msg.Body = []byte("Hey Pushpa!!")
				mxdxs.Send(msg)
			}
		}()

		go func() {
			mxdxs := mxdx.GetChannel("Sample3")
			for message := range mxdxs.In {
				fmt.Printf("Message:%v", message.String())
				msg := new(Message)
				id, _ := uuid.NewV4()
				msg.Id = id.String()
				msg.Body = []byte("Me tooo!!")
				mxdxs.Send(msg)
			}
		}()
		for {
			time.Sleep(10 * time.Minute)
		}

	}

}

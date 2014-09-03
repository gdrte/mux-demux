package muxdemux

import (
	"fmt"
	uuid "github.com/nu7hatch/gouuid"
	. "gopkg.in/check.v1"
	"net"
	"testing"
	"time"
)

var multiline = `
// Hook up gocheck into the "go test" runner.

`

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
						if mxdxs, err := mxdx.GetChannel("Sample"); err != NotAvailable {
							msg := new(Message)
							id, _ := uuid.NewV4()
							msg.Id = id.String()
							msg.Body = []byte(multiline)
							mxdxs.Send(msg)
							time.Sleep(1 * time.Second)
						} else {
							return
						}
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
						if mxdxs, err := mxdx.GetChannel("Sample2"); err != NotAvailable {
							msg := new(Message)
							id, _ := uuid.NewV4()
							msg.Id = id.String()
							msg.Body = []byte("Hey Deva!!")
							mxdxs.Send(msg)
							time.Sleep(1 * time.Second)
						} else {
							return
						}
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
						if mxdxs, err := mxdx.GetChannel("Sample3"); err != NotAvailable {
							msg := new(Message)
							id, _ := uuid.NewV4()
							msg.Id = id.String()
							msg.Body = []byte("I am from Anantapuram!!")
							mxdxs.Send(msg)
							time.Sleep(1 * time.Second)
						} else {
							return
						}
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
			time.Sleep(20 * time.Second)
			return
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
		time.Sleep(2 * time.Second)
		go func() {
			if mxdxs, err := mxdx.GetChannel("Sample"); err != NotAvailable {
				for message := range mxdxs.In {
					fmt.Printf("\nMessage: %s", string(message.Body))
					msg := new(Message)
					id, _ := uuid.NewV4()
					msg.Id = id.String()
					msg.Body = []byte("I am fine, Howdy!!")
					mxdxs.Send(msg)
				}
			} else {
				return
			}
		}()

		go func() {
			if mxdxs, err := mxdx.GetChannel("Sample2"); err != NotAvailable {
				for message := range mxdxs.In {
					fmt.Printf("\nMessage: %s", string(message.Body))
					msg := new(Message)
					id, _ := uuid.NewV4()
					msg.Id = id.String()
					msg.Body = []byte("Hey Pushpa!!")
					mxdxs.Send(msg)
				}
			} else {
				return
			}
		}()

		go func() {
			if mxdxs, err := mxdx.GetChannel("Sample2"); err != NotAvailable {

				for message := range mxdxs.In {
					fmt.Printf("\nMessage: %s", string(message.Body))

					msg := new(Message)
					id, _ := uuid.NewV4()
					msg.Id = id.String()
					msg.Body = []byte("Me tooo!!")
					mxdxs.Send(msg)
				}
			} else {
				return
			}
		}()
		for {
			time.Sleep(5 * time.Second)
			// mxdx.CloseChannel("Sample")
			// mxdx.CloseChannel("Sample2")
			// mxdx.CloseChannel("Sample3")
			mxdx.Close()
			time.Sleep(10 * time.Second)

			return
		}

	}

}

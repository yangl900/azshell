package main

import (
	"fmt"
)

func main() {
	uri, err := RequestCloudShell()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(uri)

	// localPort := flag.Int("port", 8002, "Local listening port.")

	// flag.Parse()

	// if !flag.Parsed() {
	// 	log.Println("Flag not parsed.")
	// 	flag.PrintDefaults()
	// 	os.Exit(1)
	// }

	// localServerHost := fmt.Sprintf("localhost:%d", *localPort)

	// ln, err := net.Listen("tcp", localServerHost)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Println("Port forwarding server up and listening on: ", localServerHost)

	// for {
	// 	conn, err := ln.Accept()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	go handleConnection(conn)
	// }
}

// func send(src net.Conn, dest *ws.Channel) {
// 	defer src.Close()
// 	buff := make([]byte, 8192)
// 	for {
// 		len, err := src.Read(buff)
// 		if err != nil {
// 			log.Println("[TCP] Failed to read socket: ", err.Error())
// 			break
// 		}
// 		log.Printf("[TCP][Received] %d bytes.\n", len)

// 		dest.Send(buff[:len])
// 		log.Printf("[WS] [Sent]     %d bytes.\n", len)
// 		log.Printf("[WS] [Send] %s", string(buff[:len]))
// 	}
// }

// func receive(src *ws.Channel, dest net.Conn) {
// 	defer dest.Close()

// 	for {
// 		buff, more := <-src.ReadChannel()
// 		if !more {
// 			log.Printf("[WS] [Closed]")
// 			break
// 		}

// 		log.Printf("[WS] [Received] %d bytes.\n", len(buff))

// 		n, err := dest.Write(buff)
// 		if err != nil {
// 			log.Printf("[TCP] Failed to write socket back: %s", err.Error())
// 			break
// 		}
// 		log.Printf("[TCP] [Sent]    %d bytes.", n)
// 		log.Printf("[TCP] [Send] %s", string(buff))
// 	}
// }

// func handleConnection(c net.Conn) {
// 	log.Println("Connection from : ", c.RemoteAddr())

// 	resp, err := getSocketURI()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	if resp.WebsocketURI == "" {
// 		log.Printf("Failed to get websocket URI. Closing connection.")
// 		c.Close()
// 		return
// 	}

// 	fmt.Println("Socket URI: ", resp.WebsocketURI)
// 	fmt.Println("Password", resp.Passowrd)

// 	wsConfig := ws.Config{
// 		ConnectRetryWaitDuration: time.Second * 1,
// 		SendReceiveBufferSize:    8192,
// 		URL: resp.WebsocketURI,
// 	}

// 	wsChan, err := ws.NewWebsocketChannel(wsConfig)
// 	if err != nil {
// 		log.Fatal(err)
// 		return
// 	}

// 	log.Println("Connected to ", resp.WebsocketURI)
// 	wsChan.Send([]byte(resp.Passowrd))

// 	log.Println("Authenticated.")

// 	// go routines to initiate bi-directional communication for local server with a
// 	// remote server
// 	go send(c, wsChan)
// 	go receive(wsChan, c)
// }

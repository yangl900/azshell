package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/pkg/term"

	"github.com/manifoldco/promptui"
	"github.com/yangl900/azshell/ws"
)

func main() {
	var tenantID string
	flag.StringVar(&tenantID, "tenant", "", "Specify the tenant Id.")
	flag.Parse()

	token, err := acquireBootstrapToken()
	if err != nil {
		fmt.Println(err)
		return
	}

	tenants, err := getTenants(token)
	if err != nil {
		fmt.Println(errors.New("Failed to list tenants: " + err.Error()))
		return
	}

	if len(tenants) == 0 {
		fmt.Println("No tenants found.")
		return
	}

	if len(tenants) == 1 && tenantID == "" {
		tenantID = tenants[0].TenantID
	}

	if len(tenants) > 1 && tenantID == "" {
		options := []string{}

		for _, t := range tenants {
			options = append(options, fmt.Sprintf("%s (%s)", t.DisplayName, t.TenantID))
		}

		prompt := promptui.Select{
			Label: "Select Tenant",
			Items: options,
		}

		index, _, err := prompt.Run()
		if err != nil {
			fmt.Println("Specify the --tenant option since multiple tenant available.")
			return
		}

		tenantID = tenants[index].TenantID
	}

	uri, err := RequestCloudShell(tenantID)
	if err != nil {
		fmt.Println(err)
	}

	wsURI, err := RequestTerminal(tenantID, uri)
	if err != nil {
		fmt.Println(err)
	}

	wsConfig := ws.Config{
		ConnectRetryWaitDuration: time.Second * 1,
		SendReceiveBufferSize:    8192,
		URL:                      wsURI,
	}

	wsChan, err := ws.NewWebsocketChannel(wsConfig)
	if err != nil {
		log.Fatal(err)
		return
	}

	stdIn, stdOut, _ := term.StdStreams()

	state, err := term.MakeRaw(os.Stdin.Fd())
	if err != nil {
		fmt.Println(err)
	}

	defer term.RestoreTerminal(os.Stdin.Fd(), state)

	go send(wsChan, stdIn)
	receive(wsChan, stdOut)
}

func send(dest *ws.Channel, stdIn io.ReadCloser) {
	buff := make([]byte, 1)
	for {
		len, err := stdIn.Read(buff)
		if err != nil {
			log.Println("Failed to read stdin: ", err.Error())
			break
		}

		dest.Send(buff[:len])
	}
}

func receive(src *ws.Channel, stdOut io.Writer) {
	for {
		buff, more := <-src.ReadChannel()
		if !more {
			log.Printf("Bye.\r\n")
			break
		}

		_, err := stdOut.Write(buff)
		if err != nil {
			log.Printf("Failed to write: %s", err.Error())
			break
		}
	}
}

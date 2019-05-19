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
	var reset, help bool
	flag.StringVar(&tenantID, "tenant", "", "Specify the tenant Id.")
	flag.BoolVar(&reset, "reset", false, "Reset the presisted tenant settings.")
	flag.BoolVar(&help, "help", false, "Show the help text.")
	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	if reset {
		err := os.Remove(defaultSettingsPath())
		if err != nil {
			log.Printf("Failed to remove settings: %v", err)
		}
		return
	}

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
		s, err := readSettings()
		if err != nil || s.ActiveTenant == "" {
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
			saveSettings(settings{ActiveTenant: tenantID})
		} else {
			tenantID = s.ActiveTenant
		}
	}

	css, err := ReadCloudShellUserSettings(tenantID)
	if err != nil {
		fmt.Println(err)
		return
	}

	if css.Properties == nil || css.Properties.StorageProfile == nil {
		fmt.Println("It seems you haven't setup your cloud shell account yet. Navigate to https://shell.azure.com to complete account setup.")
		return
	}

	uri, err := RequestCloudShell(tenantID)
	if err != nil {
		fmt.Println(err)
		return
	}

	t, err := RequestTerminal(tenantID, uri, css.Properties.PreferredShellType)
	if err != nil || t.SocketURI == "" {
		fmt.Println("Failed to connect to cloud shell terminal.", err)
		return
	}

	wsConfig := ws.Config{
		ConnectRetryWaitDuration: time.Second * 1,
		SendReceiveBufferSize:    8192,
		URL: t.SocketURI,
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

	go monitorSize(t)

	defer term.RestoreTerminal(os.Stdin.Fd(), state)

	go send(wsChan, stdIn)
	receive(wsChan, stdOut)
}

func monitorSize(t *Terminal) {
	curSize := &term.Winsize{}
	for {
		size, err := term.GetWinsize(os.Stdin.Fd())

		if err == nil {
			if curSize.Height != size.Height || curSize.Width != size.Width {
				curSize = size
				t.Resize(curSize)
			}
		}

		time.Sleep(time.Second * 1)
	}
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

package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/pilebones/go-udev/netlink"
)

var (
	vendorId          *string = flag.String("vendorid", "", "Vendor ID of monitored usb device")
	modelId           *string = flag.String("modelid", "", "Product ID of monitored usb device")
	connectCommand    *string = flag.String("connect", "", "Command to execute on connect. Will be run with 'sh -c'")
	disconnectCommand *string = flag.String("disconnect", "", "Command to execute on disconnect. Will be run with 'sh -c'")
)

func main() {
	flag.Parse()

	_ar := "add|remove"
	filt := netlink.RuleDefinition{
		Action: &_ar,
		Env: map[string]string{
			"SUBSYSTEM": "input",
		},
	}
	if vendorId != nil {
		filt.Env["ID_VENDOR_ID"] = *vendorId
	}
	if modelId != nil {
		filt.Env["ID_MODEL_ID"] = *modelId
	}

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		log.Fatalf("Unable to connect ot Netlink Kobject Uevent socket: %s", err)
	}
	defer conn.Close()

	queue := make(chan netlink.UEvent)
	dqueue := debounce(1*time.Second, queue)
	errors := make(chan error)
	mon := conn.Monitor(queue, errors, &filt)

	// Signal handling
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-signals
		log.Println("Signal received. Exiting")
		close(mon)
		os.Exit(0)
	}()

	// Handle messages
	for {
		select {
		case uevent, ok := <-dqueue:
			if !ok {
				log.Println("Event stream closed. Bye")
				close(mon)
				os.Exit(0)
			}
			go handleEvent(uevent)
		case err := <-errors:
			log.Println("ERROR:", err)
		}
	}
}

func handleEvent(ev netlink.UEvent) {
	var cmd *string

	log.Printf("Matching %s event received", ev.Action)
	switch ev.Action {
	case netlink.ADD:
		cmd = connectCommand
	case netlink.REMOVE:
		cmd = disconnectCommand
	}
	if cmd == nil {
		log.Printf("No command configured. Ignoring")
		return
	}

	log.Printf("Executing: %s", *cmd)
	out_b, err := exec.Command("/bin/sh", "-c", *cmd).CombinedOutput()
	if err != nil {
		log.Printf("ERROR: %s", out_b)
	}
	if string(out_b) != "" {
		log.Printf("Command output: %s", out_b)
	}
}

func debounce(interval time.Duration, input chan netlink.UEvent) chan netlink.UEvent {
	output := make(chan netlink.UEvent)
	go func() {
		enabled := true
		for {
			select {
			case ev, ok := <-input:
				if !ok {
					// Channel closed, we're done
					close(output)
					return
				}
				if enabled {
					output <- ev
					enabled = false
				}

			case <-time.After(interval):
				enabled = true
			}
		}
	}()
	return output
}

package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pilebones/go-udev/netlink"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	configFile *string
	rulesJson  *string
	debug      bool
)

func init() {
	viper.SetConfigName("display-switch")
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		cfg = os.Getenv("HOME") + "/.config"
	}
	viper.AddConfigPath(cfg + "/display-switch/")
	viper.AddConfigPath("/etc/display-switch/")
	viper.AddConfigPath(".")
	configFile = flag.StringP("config", "c", "", "Pathname of configuration file")
	rulesJson = flag.String("rules", "", "JSON string defining device event matching rules")
	flag.StringP("add-command", "a", "", "Command to run when matching device is connected/added")
	flag.StringP("remove-command", "r", "", "Command to run when matching device is disconnected/removed")
	flag.BoolVarP(&debug, "debug", "d", false, "Print extra debugging information")
	dur, _ := time.ParseDuration("500ms")
	flag.Duration("debounce-window", dur, "How long to wait after an event before processing more events")
	flag.CommandLine.SetNormalizeFunc(func(f *flag.FlagSet, name string) flag.NormalizedName {
		name = strings.Replace(name, "-", "_", -1)
		return flag.NormalizedName(name)
	})
}

func main() {
	flag.Parse()
	if configFile != nil {
		viper.SetConfigFile(*configFile)
	}
	viper.BindPFlags(flag.CommandLine)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Could not read config file: %s", err)
	}

	var filt netlink.RuleDefinitions
	if rulesJson != nil && *rulesJson != "" {
		if debug {
			log.Println("Parsing rules:", viper.GetString("rules"))
		}
		if err := json.Unmarshal([]byte(*rulesJson), &filt.Rules); err != nil {
			log.Fatalf("Could not parse match rules: %s. Bailing out.", err)
			os.Exit(1)
		}
	} else if err := viper.UnmarshalKey("rules", &filt.Rules); err != nil {
		log.Fatalf("Could not parse match rules: %s. Bailing out.", err)
		os.Exit(1)
	}
	if debug {
		log.Printf("Rules: %v", filt)
	}

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		log.Fatalf("Unable to connect to Netlink Kobject Uevent socket: %s", err)
	}
	defer conn.Close()

	reset := make(chan bool)
	queue := make(chan netlink.UEvent)
	dqueue := debounce(viper.GetDuration("debounce_window"), queue, reset)
	if debug {
		log.Printf("Using %s as the debounce window", viper.GetDuration("debounce_window"))
	}
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
			go handleEvent(uevent, reset)
		case err := <-errors:
			log.Println("ERROR:", err)
		}
	}
}

func handleEvent(ev netlink.UEvent, reset chan bool) {
	var cmd string

	log.Printf("Matching %s event received", ev.Action)
	switch ev.Action {
	case netlink.ADD:
		cmd = viper.GetString("add_command")
	case netlink.REMOVE:
		cmd = viper.GetString("remove_command")
	}
	if cmd == "" {
		log.Printf("No command configured. Ignoring")
		reset <- true
		return
	}

	log.Printf("Executing: %s", cmd)
	out_b, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		log.Printf("ERROR: %s", err)
	}
	if string(out_b) != "" {
		log.Printf("Command output: %s", out_b)
	}
}

func debounce(interval time.Duration, input chan netlink.UEvent, reset chan bool) chan netlink.UEvent {
	output := make(chan netlink.UEvent)
	go func() {
		enabled := true
		do_reset := func() {
			if !enabled && debug {
				log.Println("Reset debounce")
			}
			enabled = true
		}
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
			case <-reset:
				do_reset()
			case <-time.After(interval):
				do_reset()
			}
		}
	}()
	return output
}

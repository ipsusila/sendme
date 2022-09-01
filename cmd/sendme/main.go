package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/ipsusila/sendme"
	"github.com/k0kubun/pp"
	"golang.org/x/term"
)

var (
	fConf       = flag.String("conf", "config.hjson", "Configuration file")
	fConfirm    = flag.Bool("confirm", false, "Confirm before send")
	fSendMode   = flag.Bool("send", false, "Sending mode, otherwise testing mode")
	fTestConfig = flag.Bool("testconf", false, "Test configuration, do not send email")
	fVerbose    = flag.Bool("verbose", false, "Verbose mode")
)

func main() {
	flag.Parse()

	start := time.Now()
	conf, err := sendme.LoadConfig(*fConf)
	if err != nil {
		log.Fatalf("Error loading configuration file `%s`: %v\n", *fConf, err)
	}

	// Check for configuration validity
	if conf.Server == nil {
		log.Fatalln("Server configuration not specified")
	}
	if conf.Delivery == nil {
		log.Fatalln("Delivery configuration not specified")
	}

	// Prompt for username
	if conf.Server.Username == "" {
		fmt.Print("Username: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			conf.Server.Username = scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error scanning input: %v\n", err)
		}

		if u := strings.TrimSpace(conf.Server.Username); u == "" {
			log.Fatalln("Username not specified")
		}
	}

	if conf.Server.Password == "" {
		// scan password
		fmt.Print("Password: ")
		bytepw, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			os.Exit(1)
		}
		conf.Server.Password = string(bytepw)
		fmt.Println()
	}

	// override config
	if *fConfirm {
		conf.Delivery.SkipConfirmBeforeSend = !*fConfirm
	}
	if !conf.Delivery.SendMode {
		conf.Delivery.SendMode = *fSendMode
	}
	if !conf.Verbose {
		conf.Verbose = *fVerbose
	}

	// Create mailer
	mailer, err := sendme.NewMailer(conf)
	if err != nil {
		log.Fatalf("Error while creating mailer: %v\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *fTestConfig {
		// Test config
		conf.Server.Username = "<username>"
		conf.Server.Password = "**********"
		pp.Println(conf)
	} else {
		if err := mailer.Send(ctx); err != nil {
			log.Fatalf("Error sending email: %v\n", err)
		}
		log.Printf("Sending email done, elapsed: %v\n", time.Since(start))
	}
}

package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/ipsusila/sendme"
)

var (
	fConf     = flag.String("conf", "config.hjson", "Configuration file")
	fConfirm  = flag.Bool("confirm", false, "Confirm before send")
	fSendMode = flag.Bool("send", false, "Sending mode, otherwise testing mode")
)

func main() {
	flag.Parse()

	start := time.Now()
	conf, err := sendme.LoadConfig(*fConf)
	if err != nil {
		log.Fatalf("Error loading configuration file `%s`: %v\n", *fConf, err)
	}

	// override config
	conf.Delivery.SkipConfirmBeforeSend = !*fConfirm
	conf.Delivery.SendMode = *fSendMode

	// Create mailer
	mailer, err := sendme.NewMailer(conf)
	if err != nil {
		log.Fatalf("Error while creating mailer: %v\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mailer.Send(ctx); err != nil {
		log.Fatalf("Error sending email: %v\n", err)
	}

	log.Printf("Sending email done, elapsed: %v\n", time.Since(start))
}

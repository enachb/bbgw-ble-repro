package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/paulbellamy/ratecounter"

	http "net/http"
	_ "net/http/pprof"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
)

var beaconRate *ratecounter.RateCounter

func deadManSwitch(beaconRate *ratecounter.RateCounter, interval time.Duration, rateInterval time.Duration) {
	// Record seen BLE devices per rateInterval. Used as dead man switch.
	ticker := time.NewTicker(interval)
	for range ticker.C {
		if beaconRate.Rate() < 1 {
			log.Fatal("Deadman switch triggered. Not seeing any beacons.")
		}
		log.Infof("seen BLE device rate per %ds: %d", rateInterval/time.Second, beaconRate.Rate())
	}
}

func advHandler(a ble.Advertisement) {
	// We found a bleutooth device
	// Hit dead man switch
	beaconRate.Incr(1)
}

func main() {

	// Parse flags
	scanInterval := flag.Duration("d", 2*time.Second, "Scanning aggregation interval. Duplicate readings will be squashed during a scan. Use standard units e.g. 2s, 1m")
	debug := flag.Bool("debug", false, "Turn on debug log level.")
	enablePprof := flag.Bool("pprof", false, "Start pprof on localhost:6060")
	bleDevice := flag.String("device", "default", "BLE device to be used for scanning.")
	deadManInt := flag.Duration("dead", 30*time.Second, "Interval at least one bluetooth device has to be seen before the process is killed.")

	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	log.SetFormatter(&log.TextFormatter{})

	// Register handlers.
	dev, err := dev.NewDevice(*bleDevice)
	if err != nil {
		log.Fatalf("can't new device : %s", err)
	}
	ble.SetDefaultDevice(dev)

	// Scan for specified durantion, or until interrupted by user.
	log.Infof("Scanning interval %s...\n", *scanInterval)

	// Setup dead man switch before we start collecting beacons to avoid race condition
	deadManRateInterval := 10 * time.Second
	beaconRate = ratecounter.NewRateCounter(deadManRateInterval)

	// Kickof aync scanning
	go func() {
		for {
			ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), *scanInterval))
			err = ble.Scan(ctx, true, advHandler, nil) // allow dupes. Will be filtered out before sending.
			switch errors.Cause(err) {
			case nil:
			// case err
			case context.DeadlineExceeded:
				log.Debug("Scanning loop done")
			case context.Canceled:
				log.Warn("Scanning loop canceled")
				log.Fatal("Received kill signal. Terminating.")
			default:
				log.Error(fmt.Errorf("err %w", err))
				log.Errorf("%+v", err)
				log.Errorf("%+v", errors.Cause(err))
				log.Fatal("Error performing BLE scan. Crashing hoping that a reboot will fix this.", err)
			}
		}
	}()

	// Start profiling if requested
	if *enablePprof {
		go func() {
			log.Info("Starting pprof profiler on localhost:606")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// Start deadman switch
	go deadManSwitch(beaconRate, *deadManInt, deadManRateInterval)

	// Blocks forever
	select {}
}

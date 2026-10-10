//go:build tinygo

// Station firmware for the original ESP32. It joins a Wi-Fi access point
// through espradio and POSTs a heartbeat to a Prometheus Pushgateway
// every 10 seconds.
package main

import (
	"strconv"
	"time"

	"tinygo.org/x/drivers/netdev"
	nl "tinygo.org/x/drivers/netlink"
	"tinygo.org/x/espradio/netlink"

	"esp32-ex1/internal/pushgateway"
)

// Injected at build time. See the Makefile and .env.example.
// pushURL defaults to the LAN gateway so a plain tinygo flash still
// targets it; ssid and password stay empty unless -ldflags sets them.
var (
	ssid         string
	password     string
	pushURL      = "http://192.168.31.102:9091"
	pushJob      = "esp32-ex1"
	pushInstance = "esp32"
)

const pushInterval = 10 * time.Second

func main() {
	// The USB-UART adapter often drops the first bytes after reset.
	time.Sleep(2 * time.Second)
	println("esp32-ex1")

	// Scroll MANTY on the OLED while the radio comes up and while the
	// push loop runs. The panel keeps its last picture across reset, so
	// this also replaces the previous "MANTY WORLD!" frame.
	go runManty(newDisplay())

	link := connectWiFi()
	if addr, err := link.Addr(); err == nil {
		println("wifi ip:", addr.String())
	}
	if mac, err := link.GetHardwareAddr(); err == nil {
		println("wifi mac:", mac.String())
	}

	endpoint, err := pushgateway.Endpoint(pushURL, pushJob, pushInstance)
	if err != nil {
		fail("push url: " + err.Error())
	}
	pusher := pushgateway.New(endpoint)
	println("push:", endpoint)

	var (
		okCount uint64
		started = time.Now()
	)
	for {
		uptime := uint64(time.Since(started) / time.Second)
		// pushTotal counts this attempt only after the gateway accepts it.
		body := pushgateway.Sample(uptime, okCount+1)
		if err := pusher.Push(body); err != nil {
			println("push err:", err.Error())
		} else {
			okCount++
			println("pushed uptime=" + strconv.FormatUint(uptime, 10) +
				"s n=" + strconv.FormatUint(okCount, 10))
		}
		time.Sleep(pushInterval)
	}
}

func connectWiFi() *netlink.Esplink {
	if ssid == "" {
		fail("wifi: SSID empty, set WIFI_SSID in .env and rebuild")
	}

	link := &netlink.Esplink{}
	netdev.UseNetdev(link)

	println("wifi: connecting to " + ssid)
	err := link.NetConnect(&nl.ConnectParams{
		Ssid:       ssid,
		Passphrase: password,
		Hostname:   "esp32-ex1",
	})
	if err != nil {
		// Enable is one-shot. A failed association cannot be retried
		// without resetting the chip.
		fail("wifi err: " + err.Error())
	}
	println("wifi: connected")
	return link
}

func fail(msg string) {
	for {
		println(msg)
		time.Sleep(time.Second)
	}
}

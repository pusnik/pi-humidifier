package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/d2r2/go-dht"
	// For controlling Orvibo stuff
	"github.com/pusnik/go-orvibo"
)

func monitorHumidity() {
	sensorType := dht.DHT22
	temperature, humidity, retried, err :=
		dht.ReadDHTxxWithRetry(sensorType, 3, false, 10)
	if err != nil {
		log.Fatal(err)
	}
	// print temperature and humidity
	fmt.Printf("Temperature = %v*C, Humidity = %v%% (retried %d times)\n",
		temperature, humidity, retried)
}

func discoverSockets() map[string]*orvibo.Device {
	//querry for sockets - print out sockets and wait for user input
	// user input to recognize the correct socket can:
	// select the socket id directly
	// says to turn off/on the socket the recognize the correct one:
	// These are our SetIntervals that run. To cancel one, simply send "<- true" to it (e.g. autoDiscover <- true)
	//var autoDiscover chan bool
	timeoutChan := time.NewTimer(time.Second * 5).C

	ready, err := orvibo.Prepare() // You ready?
	if ready == true {
		fmt.Println("Searching...")

		// Look for new devices every minute
		//autoDiscover = setInterval(orvibo.Discover, time.Minute)
		// Resubscription should happen every 5 minutes, but we make it 3, just to be on the safe side
		//resubscribe = setInterval(orvibo.Subscribe, time.Minute*3)
		orvibo.Discover() // Discover all sockets

		//start go routine to check for messages
		go func() {
			for {
				orvibo.CheckForMessages()
			}
		}()

		for { // Loop forever
			select { // This lets us do non-blocking channel reads. If we have a message, process it. If not, check for UDP data and loop
			case <-timeoutChan:
				orvibo.Close()
				return orvibo.Devices
			case msg := <-orvibo.Events:
				switch msg.Name {
				case "socketfound":
					fmt.Println("---------------")
					fmt.Println("Found:", msg.DeviceInfo.IP)
					fmt.Println("MAC:", msg.DeviceInfo.MACAddress)
				}
			}
		}
	} else {
		fmt.Println("Error:", err)
	}
	return orvibo.Devices
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("What can I do for you?\n")

	//add states to always know in what state are we in
	//WAITING_FOR_INPUT, DISCOVERING, MONITORING

	for { // Loop forever
		command, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		command = strings.ToLower(strings.TrimSpace(command))

		if command == "discover" {
			discoverSockets()
		} else if command == "exit" || command == "no" {
			os.Exit(3)
		} else if command == "monitor" {
			monitorHumidity()
		} else {
			fmt.Println("Unknown command!")
		}
		fmt.Println("Done! Do you want anything else?")
	}
}

func setInterval(what func(), delay time.Duration) chan bool {
	stop := make(chan bool)

	go func() {
		for {
			what()
			select {
			case <-time.After(delay):
			case <-stop:
				return
			}
		}
	}()

	return stop
}

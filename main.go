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

//  States we can be in at any point in time
type stateT int

const (
	stateSTART                = stateT(iota)
	stateCONFIG               // start configuration
	stateCONFIGSCAN           // scan for smart sockets
	stateCONFIGSETSOCKET      // set which socket do we want to control
	stateCONFIGSETTHRESHOLD   // set thresholds for humidifier
	stateMONITOR              // in monitor mode
	stateMONITORREADTEMPHUMID // read values from socket
	stateMONITORSAVETODB      // save data to db to analize
	stateMONITORCONTROLSOCKET // turn on/off the socket
)

type piHumidifierInterface interface {
	config()
}

type measurement struct {
	temperature int
	humidity    int
	datetime    time
}

//PiHumidifier is our main class
type PiHumidifier struct {
	state           stateT //  Current state
	socket          *orvibo.Device
	devices         map[string]*orvibo.Device
	humidityLow     int
	humidityHigh    int
	lastMeasurement *measurement
}

func () monitorHumidity() {
	sensorType := dht.DHT22
	temperature, humidity, retried, err :=
		dht.ReadDHTxxWithRetry(sensorType, 3, false, 10)
	if err != nil {
		log.Fatal(err)
	}
	// print temperature and humidity
	return &measurement{temperature: temperature, humidity: humidity, datetime: Now()}
}

func (hum *PiHumidifier) config() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("What can I do for you?\n")

	for { // Loop forever
		command, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		command = strings.ToLower(strings.TrimSpace(command))

		if command == "discover" {
			fmt.Println("Discover")
			hum.state = stateCONFIGSCAN
			return
		} else if command == "exit" || command == "no" {
			os.Exit(3)
		} else if command == "check" {
			hum.state = stateMONITORREADTEMPHUMID
			return
		} else {
			fmt.Println("Unknown command!")
		}
		fmt.Println("Done! Do you want anything else?")
	}
}

func discoverSockets() map[string]*orvibo.Device {
	timeoutChan := time.NewTimer(time.Second * 5).C

	ready, err := orvibo.Prepare() // You ready?
	if ready == true {
		fmt.Println("Searching...")

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

func (hum *PiHumidifier) executeFsm() {
	fmt.Println("In FSM")
	for {
		switch hum.state {
		case stateSTART:
			fmt.Println("State Start")
			hum.config()
		case stateCONFIGSCAN:
			hum.devices = discoverSockets()
			hum.state = stateSTART
		case stateMONITORREADTEMPHUMID:
			fmt.Println("State monitor")
			hum.lastMeasurement = monitorHumidity()
			fmt.Println(hum.lastMeasurement)
			hum.state = stateSTART
		}
	}
}

func main() {
	fmt.Println("Starting")
	hum := &PiHumidifier{humidityHigh: 60, humidityLow: 35, state: stateSTART, socket: nil}
	hum.executeFsm()
}

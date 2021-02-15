package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/goburrow/modbus"
	"github.com/zathras777/modbusdev"
)

var (
	cfgFn           string
	cfg             *configData
	err             error
	asDaemon        bool
	inverterAddress string
	mqttClient      mqtt.Client
)

func main() {
	flag.BoolVar(&asDaemon, "daemon", false, "Run as daemon? [default false]")
	flag.StringVar(&cfgFn, "cfg", "configuration.yaml", "Configuration file to parse for database settings")
	flag.Parse()

	cfg, err = parseConfiguration(cfgFn)
	if err != nil {
		log.Fatalf("Unable to parse configuration file %s. Exiting.\n%s", cfgFn, err)
	}

	inverterAddress = net.JoinHostPort(cfg.Inverter.Ipaddress, cfg.Inverter.Port)
	mqttAddress := fmt.Sprintf("tcp://%s:%d", cfg.MQTT.Host, cfg.MQTT.Port)

	if asDaemon {
		log.Printf("Communicating with inverter @ %s, data to MQTT @ %s", inverterAddress, mqttAddress)

		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigs
			done <- true
		}()

		mqOpts := mqtt.NewClientOptions()
		mqOpts.AddBroker(mqttAddress)
		mqttClient = mqtt.NewClient(mqOpts)

		hassAdvertise(mqttClient)
		if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
			log.Printf("Unable to connect to the MQTT server on %s:%d: %v", cfg.MQTT.Host, cfg.MQTT.Port, token.Error())
			return
		}
		hassAdvertise(mqttClient)

		client := modbus.TCPClient(inverterAddress)
		rdr, err := modbusdev.NewReader(client, cfg.Inverter.Type)
		if err != nil {
			log.Fatalf("Error getting a modbusdev Reader. %s", err)
		}
		log.Printf("Connected to inverter @ %s", inverterAddress)

		go func() {
			for {
				rdrMap := rdr.Map(true)
				if len(rdrMap) > 0 {
					for _, fld := range cfg.Fields {
						token := mqttClient.Publish(fld.topic, cfg.MQTT.QoS, true, fmt.Sprintf("%.02f", rdrMap[fld.Code].Ieee32))
						token.Wait()
						if token.Error() != nil {
							log.Printf("Error publishing %s: %v", fld.Register.Description, token.Error())
						}
					}
				}
				time.Sleep(time.Second * time.Duration(cfg.MQTT.Interval))
			}
		}()

		<-done
	} else {
		client := modbus.TCPClient(inverterAddress)
		rdr, err := modbusdev.NewReader(client, cfg.Inverter.Type)
		if err != nil {
			log.Fatalf("Error getting a modbusdev Reader. %s", err)
		}
		rdr.Dump(true)
	}
}

// Manager Fault - 512 => CT/Meter Fault

func hassAdvertise(client mqtt.Client) error {
	type hassAdvert struct {
		Name              string `json:"name"`
		UniqueID          string `json:"unique_id"`
		Icon              string `json:"icon,omitempty"`
		StateTopic        string `json:"state_topic"`
		UnitOfMeasurement string `json:"unit_of_measurement,omitempty"`
	}
	for _, fld := range cfg.Fields {
		haData := hassAdvert{
			Name:              fmt.Sprintf("%s %s", cfg.Name, fld.Register.Description),
			StateTopic:        fld.topic,
			UniqueID:          fmt.Sprintf("%s_%d", cfg.Name, fld.Code),
			UnitOfMeasurement: fld.Register.Units}
		switch fld.Register.Units {
		case "W":
			haData.Icon = "hass:flash"
		case "C":
			haData.Icon = "hass:thermometer"
		}
		jsonBytes, err := json.Marshal(haData)
		if err != nil {
			log.Printf("Unable to encode HA json: %s", err)
			continue
		}
		client.Publish(fmt.Sprintf("%s/sensor/%s/%d/config", cfg.MQTT.HassdiscoveryPrefix,
			cfg.Name, fld.Code), cfg.MQTT.QoS, true, jsonBytes)
	}
	return nil
}

package main

import (
	"fmt"
	"io/ioutil"
	"log"

	modbusdev "github.com/zathras777/modbusdev"
	"gopkg.in/yaml.v2"
)

type mqttData struct {
	Host                string
	Port                int
	QoS                 byte
	Interval            int
	TopicPrefix         string `yaml:"topic_prefix"`
	HassdiscoveryPrefix string
}

type inverterData struct {
	Ipaddress string
	Port      string
	Type      string
}

type fieldData struct {
	Name     string
	Code     int
	Register modbusdev.Register
	topic    string
}

type configData struct {
	Name     string
	MQTT     mqttData
	Inverter inverterData
	Fields   []fieldData
}

func parseConfiguration(cfgFn string) (*configData, error) {
	parsedCfg := configData{}
	parsedCfg.Inverter.Port = "502"

	cfgData, err := ioutil.ReadFile((cfgFn))
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(cfgData, &parsedCfg)
	if err != nil {
		return nil, err
	}

	regMap, err := modbusdev.RegistersByName(parsedCfg.Inverter.Type)
	if err != nil {
		return nil, err
	}
	var newFields []fieldData
	for _, fld := range parsedCfg.Fields {
		reg, ck := regMap[fld.Code]
		if !ck {
			log.Printf("Unable to find a register of code %d", fld.Code)
			continue
		}
		fld.Register = reg
		fld.topic = fmt.Sprintf("%s/%s/%d/state", parsedCfg.MQTT.TopicPrefix, parsedCfg.Name, fld.Code)
		newFields = append(newFields, fld)
	}
	parsedCfg.Fields = newFields
	return &parsedCfg, nil
}

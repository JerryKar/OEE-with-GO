package main

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Payload struct {
	ProductionData struct {
		PlannedProductionTime float64 `json:"PlannedProductionTime"`
		OperatingTime         float64 `json:"OperatingTime"`
		PerformanceDefinition float64 `json:"PerformanceDefinition"` // Assuming this is the same as IdealCycleTime
		TotalUnitsProduced    int     `json:"TotalUnitsProduced"`
		GoodUnitsProduced     int     `json:"GoodUnitsProduced"`
	} `json:"productionData"`
}
type OEE struct {
	Availability    float64
	Performance     float64
	Quality         float64
	Scrap           float64
	ProduceQuantity float64
	OEE             float64
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	var payload Payload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		fmt.Println("Error parsing data:", err)
		return
	}

	// Calculate OEE
	oee := CalculateOEE(payload)
	fmt.Printf("Availability:	 %.2f\n", oee.Availability)
	fmt.Printf("Performance:	 %.2f\n", oee.Performance)
	fmt.Printf("Quality:	 %.2f\n", oee.Quality)
	fmt.Printf("Scrap: 		 %.2f\n", oee.Scrap)
	fmt.Printf("ProduceQuantity: %.2f\n", oee.ProduceQuantity)
	fmt.Printf("Real-Time OEE: 	 %.2f\n", oee.OEE)
}

func main() {
	initializationProgressComplete := make(chan bool)
	go func() {

		fmt.Println("Initializing real time OEE...")
		fmt.Println("Configuring MQTT...")
		opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
		opts.SetClientID("oee-calculator")
		opts.SetDefaultPublishHandler(messagePubHandler)
		opts.SetAutoReconnect(true)

		opts.OnConnectionLost = func(client mqtt.Client, err error) {
			fmt.Printf("Connection lost: %v\n", err)
		}
		opts.OnReconnecting = func(client mqtt.Client, opts *mqtt.ClientOptions) {
			fmt.Println("Reconnecting to MQTT broker...")
		}
		opts.OnConnect = func(client mqtt.Client) {
			fmt.Println("Connected to MQTT broker")
			if token := client.Subscribe("factory/data", 1, nil); token.Wait() && token.Error() != nil {
				panic(token.Error())
			} else {
				fmt.Println("Subscribed to topic: factory/data")
			}
		}

		client := mqtt.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

		if token := client.Subscribe("factory/data", 1, nil); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

		initializationProgressComplete <- true
	}()

	select {
	case value := <-initializationProgressComplete:
		if value {
			fmt.Println("Initialization Completed")
		}

	}

	for {
		time.Sleep(1 * time.Second)
	}
}

func CalculateOEE(payload Payload) OEE {
	data := payload.ProductionData
	availability := data.OperatingTime / data.PlannedProductionTime
	idealProduction := data.OperatingTime / data.PerformanceDefinition
	performance := float64(data.TotalUnitsProduced) / idealProduction
	quality := float64(data.GoodUnitsProduced) / float64(data.TotalUnitsProduced)
	scrap := float64(data.TotalUnitsProduced) - float64(data.GoodUnitsProduced)
	producedQuantity := float64(data.GoodUnitsProduced)
	oee := availability * performance * quality

	return OEE{
		Availability:    availability,
		Performance:     performance,
		Quality:         quality,
		Scrap:           scrap,
		ProduceQuantity: producedQuantity,
		OEE:             oee,
	}
}

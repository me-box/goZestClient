package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	zest ".."
)

func main() {
	ServerKey := flag.String("server-key", "vl6wu0A@XP?}Or/&BR#LSxn>A+}L)p44/W[wXL3<", "Set the curve server key")
	Path := flag.String("path", "/kv/foo", "Set the uri path for POST and GET")
	Token := flag.String("token", "", "Set set access token")
	Payload := flag.String("payload", "{\"name\":\"dave\", \"age\":30}", "Set the uri path for POST and GET")
	ReqEndpoint := flag.String("request-endpoint", "tcp://127.0.0.1:5555", "set the request/reply endpoint")
	DealerEndpoint := flag.String("router-endpoint", "tcp://127.0.0.1:5556", "set the router/dealer endpoint")
	Mode := flag.String("mode", "OBSERVE", "set the mode of operation")
	Format := flag.String("format", "JSON", "text, json, binary to set the message content type")
	ObserveMode := flag.String("observe-mode", "data", `"data", "audit", "notification"`)
	Logging := flag.Bool("enable-logging", false, "output debug information")
	flag.Parse()

	zestC, clientErr := zest.New(*ReqEndpoint, *DealerEndpoint, *ServerKey, *Logging)
	if clientErr != nil {
		fmt.Println("Error creating client: ", clientErr.Error())
		os.Exit(2)
	}
	switch strings.ToUpper(*Mode) {
	case "POST":
		_, err := zestC.Post(*Token, *Path, []byte(*Payload), *Format)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("created")
		zestC.Close()
	case "GET":
		value, err := zestC.Get(*Token, *Path, *Format)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(string(value))
		zestC.Close()

	case "DELETE":
		err := zestC.Delete(*Token, *Path, *Format)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("deleted")
		zestC.Close()

	case "OBSERVE":
		obsTypes := map[string]zest.ObserveMode{
			"data":         zest.ObserveModeData,
			"audit":        zest.ObserveModeAudit,
			"notification": zest.ObserveModeNotification,
		}

		if val, ok := obsTypes[*ObserveMode]; ok {
			dataChan, doneChan, obsErr := zestC.Observe(*Token, *Path, *Format, val, 0)
			if obsErr != nil {
				fmt.Println(" Error: ", obsErr.Error())
				break
			}

			fmt.Println("Blocking waiting for data on chan ", dataChan)
			for resp := range dataChan {
				fmt.Println("Value returned from observer: ", string(resp))
				doneChan <- 1
			}
		} else {
			fmt.Println("Unsupported observe mode ")
		}
		zestC.Close()
	case "NOTIFY":
		dataChan, doneChan, obsErr := zestC.Notify(*Token, *Path, *Format, 0)
		if obsErr != nil {
			fmt.Println(" Error: ", obsErr.Error())
			break
		}
		fmt.Println("Blocking waiting for data on Notify chan ", dataChan, " Error: ", obsErr)
		resp := <-dataChan
		fmt.Println("Value returned from notifyer: ", string(resp))
		doneChan <- 1
		zestC.Close()
	case "TEST":

		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":91}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":92}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":93}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":94}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":95}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":96}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":97}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":98}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":99}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":100}"), *Format)

		value, err := zestC.Get(*Token, *Path+"/latest", *Format)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(string(value))
		zestC.Close()

	case "TESTWRITE":
		i := 0
		for {
			data, err := zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":"+strconv.Itoa(i)+"}"), *Format)
			if err != nil {
				fmt.Println("[ERROR] " + err.Error())
			} else {
				fmt.Println("[DATA] " + string(data))
			}
			time.Sleep(time.Millisecond * 20)
			i += 1
		}
		zestC.Close()

	case "TESTREAD":
		for {
			data, err := zestC.Get(*Token, *Path, *Format)
			if err != nil {
				fmt.Println("[ERROR] " + err.Error())
			} else {
				fmt.Println("[DATA] " + string(data))
			}
			time.Sleep(time.Millisecond * 10)
		}
		zestC.Close()

	default:
		fmt.Println("Unknown method try GET,POST,OBSERVE or NOTIFY")
	}

}

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	zest "github.com/toshbrown/goZestClient"
)

func main() {
	ServerKey := flag.String("server-key", "vl6wu0A@XP?}Or/&BR#LSxn>A+}L)p44/W[wXL3<", "Set the curve server key")
	Path := flag.String("path", "/kv/foo", "Set the uri path for POST and GET")
	Token := flag.String("token", "", "Set set access token")
	Payload := flag.String("payload", "{\"name\":\"dave\", \"age\":30}", "Set the uri path for POST and GET")
	ReqEndpoint := flag.String("request-endpoint", "tcp://127.0.0.1:5555", "set the request/reply endpoint")
	DealerEndpoint := flag.String("router-endpoint", "tcp://127.0.0.1:5556", "set the router/dealer endpoint")
	Mode := flag.String("method", "OBSERVE", "set the mode of operation")
	Format := flag.String("format", "JSON", "text, json, binary to set the message content type")
	Logging := flag.Bool("enable-logging", false, "output debug information")
	flag.Parse()

	zestC, clientErr := zest.New(*ReqEndpoint, *DealerEndpoint, *ServerKey, *Logging)
	if clientErr != nil {
		fmt.Println("Error creating client: ", clientErr.Error())
		os.Exit(2)
	}
	switch strings.ToUpper(*Mode) {
	case "POST":
		err := zestC.Post(*Token, *Path, []byte(*Payload), *Format)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("created")
	case "GET":
		value, err := zestC.Get(*Token, *Path, *Format)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(string(value))
	case "OBSERVE":
		dataChan, obsErr := zestC.Observe(*Token, *Path, *Format)
		if obsErr != nil {
			fmt.Println(obsErr.Error())
		}

		fmt.Println("Blocking waiting for data on chan ", dataChan)
		resp := <-dataChan
		fmt.Println("Value returned from observer: ", string(resp))
	case "TEST":
		postErr := zestC.Post(*Token, *Path+"/at/1510747972884", []byte(*Payload), *Format)
		if postErr != nil {
			fmt.Println(postErr.Error())
		}
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":31}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":32}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":33}"), *Format)
		zestC.Post(*Token, *Path, []byte("{\"name\":\"dave\", \"age\":34}"), *Format)
		value, err := zestC.Get(*Token, *Path+"/latest", *Format)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(string(value))
	default:
		fmt.Println("Unknown method try GET,POST or OBSERVE")
	}

}

package main

import (
	"flag"
	"fmt"
	"strings"

	zest "github.com/toshbrown/goZestClient"
)

func main() {
	fmt.Println("Starting client")

	ServerKey := flag.String("server-key", "vl6wu0A@XP?}Or/&BR#LSxn>A+}L)p44/W[wXL3<", "Set the curve server key")
	Path := flag.String("path", "/kv/foo", "Set the uri path for POST and GET")
	Token := flag.String("token", "", "Set set access token")
	Payload := flag.String("payload", "{\"name\":\"dave\", \"age\":30}", "Set the uri path for POST and GET")
	ReqEndpoint := flag.String("request-endpoint", "tcp://127.0.0.1:5555", "set the request/reply endpoint")
	DealerEndpoint := flag.String("router-endpoint", "tcp://127.0.0.1:5556", "set the router/dealer endpoint")
	Mode := flag.String("method", "OBSERVE", "set the mode of operation")
	flag.Parse()

	zestC := zest.New(*ReqEndpoint, *DealerEndpoint, *ServerKey)
	switch strings.ToUpper(*Mode) {
	case "POST":
		err := zestC.Post(*ReqEndpoint, *Token, *Path, *Payload)
		if err != nil {
			fmt.Println(err.Error())
		}
	case "GET":
		value, err := zestC.Get(*ReqEndpoint, *Token, *Path)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("Value returned: ", value)
	case "OBSERVE":
		dataChan, obsErr := zestC.Observe(*ReqEndpoint, *Token, *Path)
		if obsErr != nil {
			fmt.Println(obsErr.Error())
		}

		fmt.Println("Blocking waiting for data on chan ", dataChan)
		resp := <-dataChan
		fmt.Println("Value returned from observer: ", string(resp.Payload))

	default:
		fmt.Println("Unknown method try GET,POST or OBSERVE")
	}

}

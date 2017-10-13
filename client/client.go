package main

import (
	"flag"
	"fmt"

	zest "github.com/toshbrown/goZestClient"
)

func main() {
	fmt.Println("Starting client")

	ServerKey := *flag.String("--server-key", "vl6wu0A@XP?}Or/&BR#LSxn>A+}L)p44/W[wXL3<", "Set the curve server key")
	Path := *flag.String("--path", "/kv/foo", "Set the uri path for POST and GET")
	Token := *flag.String("--token", "", "Set set access token")
	Payload := *flag.String("--payload", "{\"name\":\"dave\", \"age\":30}", "Set the uri path for POST and GET")
	ReqEndpoint := *flag.String("--request-endpoint", "tcp://127.0.0.1:5555", "set the request/reply endpoint")
	flag.Parse()

	zestC := zest.New(ReqEndpoint, ServerKey)
	err := zestC.Post(ReqEndpoint, Token, Path, Payload)
	if err != nil {
		fmt.Println(err.Error())
	}

	value, err := zestC.Get(ReqEndpoint, Token, Path)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("Value returned: ", value)

	obsErr := zestC.Observe(ReqEndpoint, Token, Path)
	if obsErr != nil {
		fmt.Println(obsErr.Error())
	}

}

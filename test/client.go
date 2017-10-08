package main

import (
	"fmt"

	zest "github.com/Toshbrown/goZestClient"
)

func main() {
	fmt.Println("Starting client")

	const ServerKey = "vl6wu0A@XP?}Or/&BR#LSxn>A+}L)p44/W[wXL3<"
	const Path = "/kv/foo"
	const Payload = "{\"name\":\"dave\", \"age\":30}"
	const ReqEndpoint = "tcp://127.0.0.1:5555"
	const toc = ""

	zestC := zest.Client{}
	zestC.Connect(ReqEndpoint, ServerKey)
	/*err := zestC.Post(ReqEndpoint, toc, Path, Payload)
	if err != nil {
		fmt.Println(err.Error())
	}*/

	value, err := zestC.Get(ReqEndpoint, toc, Path)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("Value returned: ", value)

}

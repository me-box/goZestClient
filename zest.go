package zest

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	zmq "github.com/pebbe/zmq4"
)

const me = "ZMQ client"

func log(msg string) {
	t := time.Now()
	fmt.Println("[", me, " ", t, "] ", msg)
}

func assertNotError(err error) {

	if err != nil {
		log("Error " + err.Error())
		panic("") //TODO make this stop gracefully
	}

}

func toBigendian(val uint16) uint16 {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, val)
	return binary.BigEndian.Uint16(buf)
}

func pack_16(i uint16) []byte {
	var b [2]byte
	b[0] = byte(i >> 8 & 0xff)
	b[1] = byte(i & 0xff)
	return b[:]
}

func unPack_16(b []byte) (uint16, error) {
	if len(b) < 2 {
		return uint16(0), errors.New("Not enough bytes to unpack")
	}
	i := binary.BigEndian.Uint16(b[:])
	return i, nil
}

func pack_32(i uint32) []byte {
	var b [4]byte
	b[0] = byte(i >> 24 & 0xff)
	b[1] = byte(i >> 16 & 0xff)
	b[2] = byte(i >> 8 & 0xff)
	b[3] = byte(i & 0xff)
	return b[:]
}

type ZestClient struct {
	ZMQsoc         *zmq.Socket
	serverKey      string
	endpoint       string
	dealerEndpoint string
}

//New returns a ZestClient connected to endpoint using serverKey as an identity
func New(endpoint string, dealerEndpoint string, serverKey string) ZestClient {

	z := ZestClient{}
	log("Connecting")
	var err error
	z.ZMQsoc, err = zmq.NewSocket(zmq.REQ)
	assertNotError(err)

	clientPublic, clientSecret, err := zmq.NewCurveKeypair()
	assertNotError(err)

	err = z.ZMQsoc.ClientAuthCurve(serverKey, clientPublic, clientSecret)
	assertNotError(err)

	z.serverKey = serverKey
	z.dealerEndpoint = dealerEndpoint
	z.endpoint = endpoint
	err = z.ZMQsoc.Connect(endpoint)
	assertNotError(err)

	return z
}

func (z ZestClient) Post(endpoint string, token string, path string, payload string) error {

	log("Posting")

	//post request
	zr := zestHeader{}
	zr.Code = 2
	zr.Token = token
	zr.Payload = payload

	//post options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	hostname, _ := os.Hostname()
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: string(pack_16(50))}) // 50 representing json

	bytes, marshalErr := zr.Marshal()
	assertNotError(marshalErr)

	_, reqErr := z.sendRequestAndAwaitResponse(bytes)
	assertNotError(reqErr)
	log("=> Created")
	return nil
}

func (z ZestClient) Get(endpoint string, token string, path string) (string, error) {

	log("Getting")

	zr := zestHeader{}
	zr.Code = 1
	zr.Token = token

	//options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	hostname, _ := os.Hostname()
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: string(pack_16(50))}) // 50 representing json

	bytes, marshalErr := zr.Marshal()
	assertNotError(marshalErr)

	resp, reqErr := z.sendRequestAndAwaitResponse(bytes)
	assertNotError(reqErr)

	return resp.Payload, nil
}

func (z ZestClient) Observe(endpoint string, token string, path string) (<-chan zestHeader, error) {

	zr := zestHeader{}
	zr.Code = 1
	zr.Token = token

	//options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	hostname, _ := os.Hostname()
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 6, Value: ""})
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: string(pack_16(50))}) // 50 representing json
	zr.Options = append(zr.Options, zestOptions{Number: 14, Value: string(pack_32(60))})
	bytes, marshalErr := zr.Marshal()
	assertNotError(marshalErr)

	fmt.Println(hex.Dump(bytes[:]))

	resp, reqErr := z.sendRequestAndAwaitResponse(bytes)
	assertNotError(reqErr)

	dataChan, err := z.readFromRouterSocket(resp.Payload)
	assertNotError(err)

	return dataChan, nil

}

func (z ZestClient) sendRequest(msg []byte) error {

	if z.ZMQsoc == nil {
		return errors.New("Connection is closed can't send data")
	}

	log("Sending request:")
	fmt.Println(hex.Dump(msg))

	_, err := z.ZMQsoc.SendBytes(msg, 0)
	assertNotError(err)

	return nil
}

func (z ZestClient) sendRequestAndAwaitResponse(msg []byte) (zestHeader, error) {

	if z.ZMQsoc == nil {
		return zestHeader{}, errors.New("Connection is closed can't send data")
	}

	log("Sending request:")
	fmt.Println(hex.Dump(msg))

	z.ZMQsoc.SendBytes(msg, 0)

	//TODO ADD TIME OUT
	resp, err := z.ZMQsoc.RecvBytes(0)
	assertNotError(err)

	parsedResp, errResp := z.handleResponse(resp)
	assertNotError(errResp)

	return parsedResp, nil
}

func (z *ZestClient) readFromRouterSocket(identity string) (<-chan zestHeader, error) {

	//TODO ADD TIME OUT
	dealer, err := zmq.NewSocket(zmq.DEALER)
	assertNotError(err)

	err = dealer.SetIdentity(identity)
	assertNotError(err)

	clientPublic, clientSecret, err := zmq.NewCurveKeypair()
	assertNotError(err)
	err = dealer.ClientAuthCurve(z.serverKey, clientPublic, clientSecret)
	assertNotError(err)

	connError := dealer.Connect(z.dealerEndpoint)
	assertNotError(connError)

	dataChan := make(chan zestHeader)
	go func(output chan<- zestHeader) {
		for {
			fmt.Println("Waiting for response on id ", identity, " .....")
			resp, err := dealer.RecvBytes(0)
			assertNotError(err)
			parsedResp, errResp := z.handleResponse(resp)
			assertNotError(errResp)

			output <- parsedResp
		}
	}(dataChan)

	return dataChan, nil
}

func (z ZestClient) handleResponse(msg []byte) (zestHeader, error) {

	log("Got response:")
	fmt.Println(hex.Dump(msg))

	zr := zestHeader{}

	err := zr.Parse(msg)
	assertNotError(err)

	fmt.Println(zr)

	switch zr.Code {
	case 65:
		//created
		return zr, nil
	case 69:
		//content
		return zr, nil
	case 128:
		return zr, errors.New("bad request")
	case 129:
		return zr, errors.New("unauthorized")
	case 143:
		return zr, errors.New("unsupported content format")
	}
	return zr, errors.New("invalid code:" + strconv.Itoa(int(zr.Code)))
}

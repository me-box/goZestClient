package zest

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	zmq "github.com/pebbe/zmq4"
)

const me = "ZMQ client"

func assertNotError(err error) {

	if err != nil {
		fmt.Println("Error " + err.Error())
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
	enableLogging  bool
}

//New returns a ZestClient connected to endpoint using serverKey as an identity
func New(endpoint string, dealerEndpoint string, serverKey string, enableLogging bool) (ZestClient, error) {

	z := ZestClient{}
	z.enableLogging = enableLogging

	z.log("Connecting")
	var err error
	z.ZMQsoc, err = zmq.NewSocket(zmq.REQ)
	if err != nil {
		return z, err
	}

	clientPublic, clientSecret, err := zmq.NewCurveKeypair()
	if err != nil {
		return z, err
	}

	err = z.ZMQsoc.ClientAuthCurve(serverKey, clientPublic, clientSecret)
	if err != nil {
		return z, err
	}
	z.serverKey = serverKey
	z.dealerEndpoint = dealerEndpoint
	z.endpoint = endpoint
	err = z.ZMQsoc.Connect(endpoint)
	if err != nil {
		return z, err
	}

	return z, nil
}

func (z ZestClient) Post(token string, path string, payload []byte, contentFormat string) error {

	z.log("Posting")

	err := checkContentFormatFormat(contentFormat)
	if err != nil {
		return err
	}

	//post request
	zr := zestHeader{}
	zr.Code = 2
	zr.Token = token
	zr.Payload = payload

	//post options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	hostname, _ := os.Hostname()
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: string(pack_16(contentFormatToInt(contentFormat)))})

	bytes, marshalErr := zr.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	_, reqErr := z.sendRequestAndAwaitResponse(bytes)
	if reqErr != nil {
		return reqErr
	}
	z.log("=> Created")
	return nil
}

func (z ZestClient) Get(token string, path string, contentFormat string) ([]byte, error) {

	z.log("Getting")

	err := checkContentFormatFormat(contentFormat)
	if err != nil {
		return nil, err
	}

	zr := zestHeader{}
	zr.Code = 1
	zr.Token = token

	//options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	hostname, _ := os.Hostname()
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: string(pack_16(contentFormatToInt(contentFormat)))})

	bytes, marshalErr := zr.Marshal()
	if marshalErr != nil {
		return bytes, marshalErr
	}

	resp, reqErr := z.sendRequestAndAwaitResponse(bytes)
	if reqErr != nil {
		return bytes, reqErr
	}

	return resp.Payload, nil
}

func (z ZestClient) Observe(token string, path string, contentFormat string) (<-chan []byte, error) {

	err := checkContentFormatFormat(contentFormat)
	if err != nil {
		return nil, err
	}

	zr := zestHeader{}
	zr.Code = 1
	zr.Token = token

	//options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	hostname, _ := os.Hostname()
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 6, Value: ""})
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: string(pack_16(contentFormatToInt(contentFormat)))})
	zr.Options = append(zr.Options, zestOptions{Number: 14, Value: string(pack_32(60))})
	bytes, marshalErr := zr.Marshal()
	if marshalErr != nil {
		return nil, marshalErr
	}

	resp, reqErr := z.sendRequestAndAwaitResponse(bytes)
	if reqErr != nil {
		return nil, reqErr
	}

	dataChan, err := z.readFromRouterSocket(resp)
	if err != nil {
		return nil, err
	}

	return dataChan, nil

}

func (z ZestClient) sendRequest(msg []byte) error {

	if z.ZMQsoc == nil {
		return errors.New("Connection is closed can't send data")
	}

	z.log("Sending request:")
	z.log("\n" + hex.Dump(msg))

	_, err := z.ZMQsoc.SendBytes(msg, 0)
	if err != nil {
		return err
	}

	return nil
}

func (z ZestClient) sendRequestAndAwaitResponse(msg []byte) (zestHeader, error) {

	if z.ZMQsoc == nil {
		return zestHeader{}, errors.New("Connection is closed can't send data")
	}

	z.log("Sending request:")
	z.log("\n" + hex.Dump(msg))

	z.ZMQsoc.SendBytes(msg, 0)

	//TODO ADD TIME OUT
	resp, err := z.ZMQsoc.RecvBytes(0)
	if err != nil {
		return zestHeader{}, err
	}

	parsedResp, errResp := z.handleResponse(resp)
	if errResp != nil {
		return zestHeader{}, errResp
	}

	return parsedResp, nil
}

func (z *ZestClient) readFromRouterSocket(header zestHeader) (<-chan []byte, error) {

	//TODO ADD TIME OUT
	dealer, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		return nil, err
	}

	err = dealer.SetIdentity(string(header.Payload))
	if err != nil {
		return nil, err
	}

	serverKey := ""
	for _, option := range header.Options {
		if option.Number == 2048 {
			serverKey = option.Value
		}
	}
	z.log("Using serverKey " + serverKey)
	clientPublic, clientSecret, err := zmq.NewCurveKeypair()
	if err != nil {
		return nil, err
	}

	err = dealer.ClientAuthCurve(serverKey, clientPublic, clientSecret)
	if err != nil {
		return nil, err
	}

	connError := dealer.Connect(z.dealerEndpoint)
	if err != nil {
		return nil, connError
	}

	dataChan := make(chan []byte)
	go func(output chan<- []byte) {
		for {
			z.log("Waiting for response on id " + string(header.Payload) + " .....")
			resp, err := dealer.RecvBytes(0)
			if err != nil {
				z.log("Error reading from dealer")
				continue
			}
			parsedResp, errResp := z.handleResponse(resp)
			if errResp != nil {
				z.log("Error decoding response from dealer")
				continue
			}
			output <- parsedResp.Payload
		}
	}(dataChan)

	return dataChan, nil
}

func (z ZestClient) handleResponse(msg []byte) (zestHeader, error) {

	z.log("Got response:")
	z.log("\n" + hex.Dump(msg))

	zr := zestHeader{}

	err := zr.Parse(msg)
	if err != nil {
		return zr, err
	}

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

func (z ZestClient) log(msg string) {
	if z.enableLogging {
		t := time.Now()
		fmt.Println("[", me, " ", t, "] ", msg)
	}
}

func checkContentFormatFormat(format string) error {

	switch strings.ToUpper(format) {
	case "TEXT":
		return nil
	case "BINARY":
		return nil
	case "JSON":
		return nil
	default:
		return errors.New("Unsupported Content format: " + format)
	}

}

func contentFormatToInt(format string) uint16 {

	switch strings.ToUpper(format) {
	case "TEXT":
		return 0
	case "BINARY":
		return 42
	case "JSON":
		return 50
	}

	return 0
}

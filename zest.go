package zest

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
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
	ZMQsocMutex    *sync.Mutex
	serverKey      string
	Endpoint       string
	DealerEndpoint string
	enableLogging  bool
	hostname       string
}

//New returns a ZestClient connected to endpoint using serverKey as an identity
func New(endpoint string, dealerEndpoint string, serverKey string, enableLogging bool) (ZestClient, error) {

	z := ZestClient{}
	z.enableLogging = enableLogging

	//cache the host name to save 10ms
	z.hostname, _ = os.Hostname()

	z.log("Connecting")
	var err error
	z.ZMQsoc, err = zmq.NewSocket(zmq.REQ)
	if err != nil {
		return z, err
	}

	z.ZMQsocMutex = &sync.Mutex{}

	clientPublic, clientSecret, err := zmq.NewCurveKeypair()
	if err != nil {
		return z, err
	}

	err = z.ZMQsoc.ClientAuthCurve(serverKey, clientPublic, clientSecret)
	if err != nil {
		return z, err
	}
	z.serverKey = serverKey
	z.DealerEndpoint = dealerEndpoint
	z.Endpoint = endpoint
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
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: z.hostname})
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

func (z ZestClient) Delete(token string, path string, contentFormat string) error {

	z.log("Deleting")

	err := checkContentFormatFormat(contentFormat)
	if err != nil {
		return err
	}

	//Delete request
	zr := zestHeader{}
	zr.Code = 4
	zr.Token = token

	//Delete options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: z.hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: string(pack_16(contentFormatToInt(contentFormat)))})

	bytes, marshalErr := zr.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	_, reqErr := z.sendRequestAndAwaitResponse(bytes)
	if reqErr != nil {
		return reqErr
	}
	z.log("=> Deleted")
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
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: z.hostname})
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

func (z ZestClient) Observe(token string, path string, contentFormat string, timeout uint32) (<-chan []byte, error) {

	err := checkContentFormatFormat(contentFormat)
	if err != nil {
		return nil, err
	}

	zr := zestHeader{}
	zr.Code = 1
	zr.Token = token

	//options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: z.hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 6, Value: ""})
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: string(pack_16(contentFormatToInt(contentFormat)))})
	zr.Options = append(zr.Options, zestOptions{Number: 14, Value: string(pack_32(timeout))})
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
	z.Hexlog(msg)

	z.ZMQsocMutex.Lock()
	_, err := z.ZMQsoc.SendBytes(msg, 0)
	z.ZMQsocMutex.Unlock()
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
	z.Hexlog(msg)

	z.ZMQsocMutex.Lock()
	_, err := z.ZMQsoc.SendBytes(msg, 0)
	if err != nil {
		z.ZMQsocMutex.Unlock()
		return zestHeader{}, err
	}

	//TODO ADD TIME OUT
	resp, RecvErr := z.ZMQsoc.RecvBytes(0)
	z.ZMQsocMutex.Unlock()
	if RecvErr != nil {
		return zestHeader{}, RecvErr
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

	connError := dealer.Connect(z.DealerEndpoint)
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
	z.Hexlog(msg)

	zr := zestHeader{}

	err := zr.Parse(msg)
	if err != nil {
		return zr, err
	}

	switch zr.Code {
	case 65:
		//created
		return zr, nil
	case 66:
		//Deleted
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
	case 163:
		return zr, errors.New("service unavailable")
	case 134:
		return zr, errors.New("not acceptable")
	case 141:
		return zr, errors.New("request entity too large")
	case 160:
		return zr, errors.New("internal server error")
	}
	return zr, errors.New("invalid code:" + strconv.Itoa(int(zr.Code)))
}

func (z ZestClient) log(msg string) {
	if z.enableLogging {
		t := time.Now()
		fmt.Println("[", me, " ", t, "] ", msg)
	}
}

func (z ZestClient) Hexlog(msg []byte) {
	if z.enableLogging {
		t := time.Now()
		fmt.Println("[", me, " ", t, "] \n", hex.Dump(msg))
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

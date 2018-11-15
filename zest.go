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
	z.ZMQsoc.SetRcvtimeo(time.Second * 10)
	z.ZMQsoc.SetConnectTimeout(time.Second * 10)

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

//Close will close the socket. Should be called once you are done with the client.
// not calling this will lead to leaking TCP connections.
func (z ZestClient) Close() error {
	return z.ZMQsoc.Close()
}

func (z ZestClient) Post(token string, path string, payload []byte, contentFormat string) ([]byte, error) {

	z.log("Posting")

	err := checkContentFormatFormat(contentFormat)
	if err != nil {
		return []byte{}, err
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
		return []byte{}, marshalErr
	}

	resp, reqErr := z.sendRequestAndAwaitResponse(bytes)
	if reqErr != nil {
		return []byte{}, reqErr
	}
	z.log("=> Created")
	return resp.Payload, nil
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

type ObserveMode string

const ObserveModeData ObserveMode = "data"
const ObserveModeAudit ObserveMode = "audit"
const ObserveModeNotification ObserveMode = "notification"

func (z ZestClient) Observe(token string, path string, contentFormat string, observeMode ObserveMode, timeout uint32) (<-chan []byte, chan int, error) {

	err := checkContentFormatFormat(contentFormat)
	if err != nil {
		return nil, nil, err
	}

	zr := zestHeader{}
	zr.Code = 1
	zr.Token = token

	//options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: z.hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 6, Value: string(observeMode)})
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: string(pack_16(contentFormatToInt(contentFormat)))})
	zr.Options = append(zr.Options, zestOptions{Number: 14, Value: string(pack_32(timeout))})
	bytes, marshalErr := zr.Marshal()
	if marshalErr != nil {
		return nil, nil, marshalErr
	}

	resp, reqErr := z.sendRequestAndAwaitResponse(bytes)
	if reqErr != nil {
		return nil, nil, reqErr
	}

	dataChan, doneChan, err := z.readFromRouterSocket(resp, "")
	if err != nil {
		return nil, nil, err
	}

	return dataChan, doneChan, nil

}

func (z ZestClient) Notify(token string, path string, contentFormat string, timeout uint32) (<-chan []byte, chan int, error) {

	err := checkContentFormatFormat(contentFormat)
	if err != nil {
		return nil, nil, err
	}

	zr := zestHeader{}
	zr.Code = 1
	zr.Token = token

	//options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: z.hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: string(pack_16(contentFormatToInt(contentFormat)))})
	zr.Options = append(zr.Options, zestOptions{Number: 14, Value: string(pack_32(timeout))})

	bytes, marshalErr := zr.Marshal()
	if marshalErr != nil {
		return nil, nil, errors.New("Zest Header Marshal " + marshalErr.Error())
	}

	resp, reqErr := z.sendRequestAndAwaitResponse(bytes)
	if reqErr != nil {
		return nil, nil, errors.New("sendRequestAndAwaitResponse " + reqErr.Error())
	}

	dataChan, doneChan, err := z.readFromRouterSocket(resp, path)
	if err != nil {
		fmt.Println(resp)
		return nil, nil, errors.New("readFromRouterSocket " + err.Error())
	}

	return dataChan, doneChan, nil

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

	respChan, errChan := RecvBytesOverChan(z.ZMQsoc)
	var resp []byte
	var recvErr error
	select {
	case err := <-errChan:
		recvErr = err
	case resp = <-respChan:
	case <-time.After(11 * time.Second):
		z.log("timeout reading from router")
	}

	z.ZMQsocMutex.Unlock()

	if recvErr != nil {
		return zestHeader{}, recvErr
	}

	parsedResp, errResp := z.handleResponse(resp)
	if errResp != nil {
		return zestHeader{}, errResp
	}

	return parsedResp, nil
}

func (z *ZestClient) readFromRouterSocket(header zestHeader, path string) (<-chan []byte, chan int, error) {

	//TODO ADD TIME OUT
	dealer, err := zmq.NewSocket(zmq.DEALER)
	dealer.SetRcvtimeo(time.Second * 10)
	dealer.SetConnectTimeout(time.Second * 10)

	if err != nil {
		return nil, nil, err
	}

	serverKey := ""
	if path != "" {
		//Notify uri_path
		err = dealer.SetIdentity(path)
		if err != nil {
			return nil, nil, errors.New("dealer.SetIdentity " + err.Error())
		}
	} else {
		//Observe
		err = dealer.SetIdentity(string(header.Payload))
		if err != nil {
			return nil, nil, errors.New("dealer.SetIdentity " + err.Error())
		}
	}

	for _, option := range header.Options {
		//set Public key
		if option.Number == 2048 {
			serverKey = option.Value
			break
		}
	}

	z.log("Using serverKey " + serverKey)
	clientPublic, clientSecret, err := zmq.NewCurveKeypair()
	if err != nil {
		return nil, nil, err
	}

	err = dealer.ClientAuthCurve(serverKey, clientPublic, clientSecret)
	if err != nil {
		return nil, nil, errors.New("ClientAuthCurve " + err.Error())
	}

	connError := dealer.Connect(z.DealerEndpoint)
	if err != nil {
		return nil, nil, errors.New("dealer.Connect " + connError.Error())
	}

	dataChan := make(chan []byte)
	doneChan := make(chan int)

	go func() {
		for {
			z.log("Waiting for response on id " + string(header.Payload) + " .....")
			respChan, errChan := RecvBytesOverChan(dealer)
			select {
			case err := <-errChan:
				if err.Error() != "resource temporarily unavailable" {
					z.log("Error reading from dealer " + err.Error())
				}
				continue
			case resp := <-respChan:
				parsedResp, errResp := z.handleResponse(resp)
				if errResp != nil {
					z.log("Error decoding response from dealer")
					continue
				}
				dataChan <- parsedResp.Payload
			case <-doneChan:
				z.log("got message on doneChan")
				close(dataChan)
				dealer.Close()
				return
			case <-time.After(11 * time.Second):
				z.log("timeout reading from dealer")
				continue
			}
		}
	}()

	return dataChan, doneChan, nil
}

func RecvBytesOverChan(soc *zmq.Socket) (chan []byte, chan error) {
	dataChan := make(chan []byte)
	errChan := make(chan error)
	go func() {
		resp, err := soc.RecvBytes(0)
		if err != nil {
			errChan <- err
			return
		}
		dataChan <- resp
		close(dataChan)
		close(errChan)
	}()

	return dataChan, errChan
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

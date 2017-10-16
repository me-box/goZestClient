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

func pack_16(i uint16) ([]byte, error) {
	var b [2]byte
	b[0] = byte(i >> 8 & 0xff)
	b[1] = byte(i & 0xff)
	return b[:], nil
}

func unPack_16(b []byte) (uint16, error) {
	if len(b) < 2 {
		return uint16(0), errors.New("Not enough bytes to unpack")
	}
	i := binary.BigEndian.Uint16(b[:])
	return i, nil
}

type ZestClient struct {
	ZMQsoc *zmq.Socket
}

//New returns a ZestClient connected to endpoint using serverKey as an identity
func New(endpoint string, serverKey string) ZestClient {

	z := ZestClient{}
	log("Connecting")
	var err error
	z.ZMQsoc, err = zmq.NewSocket(zmq.REQ)
	assertNotError(err)
	clientPublic, clientSecret, err := zmq.NewCurveKeypair()
	assertNotError(err)

	err = z.ZMQsoc.ClientAuthCurve(serverKey, clientPublic, clientSecret)
	assertNotError(err)

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
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: "2"}) // 2 is ascii equivalent of 50 representing json

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
	zr.Options = append(zr.Options, zestOptions{Number: 12, Value: "2"}) // 2 is ascii equivalent of 50 representing json

	bytes, marshalErr := zr.Marshal()
	assertNotError(marshalErr)

	resp, reqErr := z.sendRequestAndAwaitResponse(bytes)
	assertNotError(reqErr)

	return resp.Payload, nil
}

func (z ZestClient) Observe(endpoint string, token string, path string) error {

	zr := zestHeader{}
	zr.Code = 1
	zr.Token = token

	//options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	hostname, _ := os.Hostname()
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: hostname})
	zr.Options = append(zr.Options, zestOptions{Number: 6, Value: ""})

	bytes, marshalErr := zr.Marshal()
	assertNotError(marshalErr)

	fmt.Println(hex.Dump(bytes[:]))

	reqErr := z.sendRequest(bytes)
	assertNotError(reqErr)

	return nil

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

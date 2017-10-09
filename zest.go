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

func pack4_16_4(i uint16, j uint16, k uint16) ([]byte, error) {
	if i > 15 {
		return nil, errors.New("i max is 4 bits")
	}
	if k > 15 {
		return nil, errors.New("k max is 4 bits")
	}
	if j > 65535 {
		return nil, errors.New("j max is 16 bits")
	}
	var b [3]byte
	b[0] = byte(i<<4 | (j >> 12 & 0xf))
	b[1] = byte(j >> 4)
	b[2] = byte(j<<4 | k)
	return b[:], nil
}

func unPack4_16_4(b []byte) (i uint16, j uint16, k uint16) {
	i = uint16(b[0] >> 4)
	var mSlice = []byte{b[0]<<4 | b[1]>>4, b[1]<<4 | b[2]>>4}
	j = binary.BigEndian.Uint16(mSlice)
	k = uint16(b[2] >> 4)
	return i, j, k
}

type Client struct {
	Client *zmq.Socket
}

func (z *Client) Connect(endpoint string, serverKey string) {

	log("Connecting")
	var err error
	z.Client, err = zmq.NewSocket(zmq.REQ)
	assertNotError(err)
	clientPublic, clientSecret, err := zmq.NewCurveKeypair()
	assertNotError(err)

	err = z.Client.ClientAuthCurve(serverKey, clientPublic, clientSecret)
	assertNotError(err)

	err = z.Client.Connect(endpoint)
	assertNotError(err)

}

func (z Client) Post(endpoint string, token string, path string, payload string) error {

	log("Posting")

	//post request
	zr := zestHeader{}
	zr.Version = 1
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

	fmt.Println(hex.Dump(bytes[:]))

	_, reqErr := z.sendRequestAndAwaitResponse(bytes)
	assertNotError(reqErr)
	log("=> Created")
	return nil
}

func (z Client) Get(endpoint string, token string, path string) (string, error) {

	zr := zestHeader{}
	zr.Version = 1
	zr.Code = 1
	zr.Token = token

	//options
	zr.Options = append(zr.Options, zestOptions{Number: 11, Value: path})
	hostname, _ := os.Hostname()
	zr.Options = append(zr.Options, zestOptions{Number: 3, Value: hostname})

	bytes, marshalErr := zr.Marshal()
	assertNotError(marshalErr)

	fmt.Println(hex.Dump(bytes[:]))

	resp, reqErr := z.sendRequestAndAwaitResponse(bytes)
	assertNotError(reqErr)
	log("=> Received")

	return resp.Payload, nil
}

func (z Client) Observe(endpoint string, token string, path string) error {

	zr := zestHeader{}
	zr.Version = 1
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

func (z Client) sendRequest(msg []byte) error {

	if z.Client == nil {
		return errors.New("Connection is closed can't send data")
	}

	log("Sending request:")
	fmt.Println(hex.Dump(msg))

	_, err := z.Client.SendBytes(msg, 0)
	assertNotError(err)

	return nil
}

func (z Client) sendRequestAndAwaitResponse(msg []byte) (zestHeader, error) {

	if z.Client == nil {
		return zestHeader{}, errors.New("Connection is closed can't send data")
	}

	log("Sending request:")
	fmt.Println(hex.Dump(msg))

	z.Client.SendBytes(msg, 0)

	//TODO ADD TIME OUT
	resp, err := z.Client.RecvBytes(0)
	assertNotError(err)

	parsedResp, errResp := z.handleResponse(resp)
	assertNotError(errResp)

	return parsedResp, nil
}

func (z Client) handleResponse(msg []byte) (zestHeader, error) {

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

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

const me = "ZMQ Test client"

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

const zestOptionsHeaderLength = 3

type zestOptions struct {
	Number uint16 //4
	len    uint16 //16
	Zxf    int    //4
	Value  string
}

func (zo *zestOptions) Marshal() ([]byte, error) {
	if zo == nil {
		return nil, errors.New("This should not be nil")
	}

	zo.len = uint16(len(zo.Value))

	//pack the header
	b, err := pack4_16_4(zo.Number, zo.len, 0xf)
	assertNotError(err)

	//copy in the value
	//TODO check value length
	b = append(b[:], zo.Value[:]...)

	return b, nil
}

func (zo *zestOptions) Parse(b []byte) error {

	return nil
}

const zestHeaderLength = 4

type zestRequest struct {
	Version uint16 //4
	tkl     uint16 //16
	oc      uint16 //4
	Code    uint16 //8
	Token   string
	Options []zestOptions
	Payload string
}

func (z *zestRequest) Marshal() ([]byte, error) {
	if z == nil {
		return nil, errors.New("This should not be nil")
	}
	//TODO check token length
	//TODO number of options < 16
	//TODO check token length

	z.oc = uint16(len(z.Options))

	//option token length must be bigendian
	z.tkl = uint16(len(z.Token))
	tklBigendian := toBigendian(z.tkl)
	fmt.Println("z.tkl ", z.tkl, "tklBigendian ", tklBigendian)
	fmt.Println("z.Code ", z.Code)
	fmt.Println("z.oc ", z.oc)

	b, err := pack4_16_4(z.Version, tklBigendian, z.oc)
	assertNotError(err)

	//copy the code and token
	b = append(b, byte(z.Code))

	if z.tkl > 0 {
		copy(b[4:], z.Token)
	}

	//append the options
	for i := 0; i < int(z.oc); i++ {
		optBytes, marshalErr := z.Options[i].Marshal()
		assertNotError(marshalErr)
		b = append(b[:], optBytes[:]...)
	}

	//add the payload
	b = append(b[:], z.Payload[:]...)

	return b, nil
}

func (z *zestRequest) Parse(msg []byte) error {

	//TODO handle options and message size
	z.Version, z.tkl, z.oc = unPack4_16_4(msg)
	z.Code = uint16(msg[3])

	return nil
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
	zr := zestRequest{}
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

	reqErr := z.sendRequest(bytes)
	assertNotError(reqErr)
	log("=> Created")
	return nil
}

func (z Client) sendRequest(msg []byte) error {

	log("Sending request:")
	fmt.Println(hex.Dump(msg))

	z.Client.SendBytes(msg, 0)

	resp, err := z.Client.RecvBytes(0)
	assertNotError(err)

	errResp := z.handleResponse(resp)
	assertNotError(errResp)

	return nil
}

func (z Client) handleResponse(msg []byte) error {

	log("Got response:")
	fmt.Println(hex.Dump(msg))

	zr := zestRequest{}

	err := zr.Parse(msg)
	assertNotError(err)

	fmt.Println(zr)

	switch zr.Code {
	case 65:
		return nil
	case 69:
		return nil
	case 128:
		return errors.New("bad request")
	case 129:
		return errors.New("unauthorized")
	case 143:
		return errors.New("unsupported content format")
	}
	return errors.New("invalid code:" + strconv.Itoa(int(zr.Code)))
}

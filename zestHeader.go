package zest

import (
	"errors"
)

type zestHeader struct {
	oc      uint8  //8
	Code    uint8  //8
	tkl     uint16 //16
	Token   string
	Options []zestOptions
	Payload string
}

func (z *zestHeader) Marshal() ([]byte, error) {
	if z == nil {
		return nil, errors.New("This should not be nil")
	}
	//TODO check token length
	//TODO number of options < 8
	//TODO check token length

	z.oc = uint8(len(z.Options))

	//option token length must be bigendian
	z.tkl = uint16(len(z.Token))
	tklBigendian := toBigendian(z.tkl)

	var b []byte
	b = append(b, byte(z.Code))
	b = append(b, byte(z.oc))
	packed, err := pack_16(tklBigendian)
	assertNotError(err)
	b = append(b, packed[:]...)

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

func (z *zestHeader) Parse(msg []byte) error {
	if len(msg) < 4 {
		return errors.New("Can't parse header not enough bytes")
	}
	//TODO handle options and message size
	z.Code = uint8(msg[0])
	z.oc = uint8(msg[1])

	z.tkl, _ = unPack_16(msg[2:4])

	if z.oc > 0 {
		//TODO handle options
	}

	//TODO start of payload is dependent on oc
	if len(msg) > 5 {
		z.Payload = string(msg[5:])
	}

	return nil
}

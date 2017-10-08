package zest

import (
	"errors"
)

type zestHeader struct {
	Version uint16 //4
	tkl     uint16 //16
	oc      uint16 //4
	Code    uint16 //8
	Token   string
	Options []zestOptions
	Payload string
}

func (z *zestHeader) Marshal() ([]byte, error) {
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

func (z *zestHeader) Parse(msg []byte) error {

	//TODO handle options and message size
	z.Version, z.tkl, z.oc = unPack4_16_4(msg)
	z.Code = uint16(msg[3])

	if z.oc > 0 {
		//TODO handle options
	}

	//TODO start of payload is dependent on oc
	z.Payload = string(msg[5:])

	return nil
}

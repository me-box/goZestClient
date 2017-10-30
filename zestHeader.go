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

	var b []byte
	b = append(b, byte(z.Code))
	b = append(b, byte(z.oc))
	packed := pack_16(z.tkl)
	b = append(b, packed[:]...)

	if z.tkl > 0 {
		b = append(b, []byte(z.Token)...)
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

	z.Code = uint8(msg[0])
	z.oc = uint8(msg[1])
	z.tkl, _ = unPack_16(msg[2:4])

	if len(msg) >= 5 {
		var remainingBytes = msg[4:]
		if z.oc > 0 {
			var err error
			for i := 0; i < int(z.oc); i++ {
				zo := zestOptions{}
				remainingBytes, err = zo.Parse(remainingBytes)
				if err != nil {
					return errors.New("Error decoding options")
				} else {
					z.Options = append(z.Options, zo)
				}

			}
		}

		if len(remainingBytes) > 0 {
			z.Payload = string(remainingBytes)
		}
	}

	return nil
}

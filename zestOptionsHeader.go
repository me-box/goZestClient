package zest

import "errors"

type zestOptions struct {
	Number uint8  //8
	len    uint16 //16
	Value  string
}

func (zo *zestOptions) Marshal() ([]byte, error) {
	if zo == nil {
		return nil, errors.New("This should not be nil")
	}

	zo.len = uint16(len(zo.Value))

	//pack the header
	var b []byte
	b = append(b, zo.Number)
	packed := pack_16(zo.len)
	b = append(b, packed[:]...)

	//copy in the value
	//TODO check value length
	b = append(b[:], zo.Value[:]...)

	return b, nil
}

func (zo *zestOptions) Parse(b []byte) ([]byte, error) {

	if len(b) < 3 {
		return nil, errors.New("Not enough bytes to Unmarshal")
	}

	zo.Number = b[0]
	zo.len, _ = unPack_16(b[1:3])
	zo.Value = string(b[4 : 4+zo.len])

	if len(b) > (4 + int(zo.len)) {
		return b[4+zo.len:], nil
	}

	return nil, nil
}

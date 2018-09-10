package zest

import "errors"

type zestOptions struct {
	Number uint16 //16
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
	packed := pack_16(zo.Number)
	b = append(b, packed[:]...)
	packed = pack_16(zo.len)
	b = append(b, packed[:]...)

	//copy in the value
	b = append(b[:], zo.Value[:]...)

	return b, nil
}

func (zo *zestOptions) Parse(b []byte) ([]byte, error) {

	if len(b) < 4 {
		return nil, errors.New("Not enough bytes to Unmarshal")
	}

	zo.Number, _ = unPack_16(b[0:2])
	zo.len, _ = unPack_16(b[2:4])
	zo.Value = string(b[4 : 4+zo.len])

	if len(b) > (4 + int(zo.len)) {
		return b[4+zo.len:], nil
	}

	return nil, nil
}

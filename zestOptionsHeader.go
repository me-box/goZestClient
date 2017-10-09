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
	packed, err := pack_16(zo.len)
	b = append(b, packed[:]...)
	assertNotError(err)

	//copy in the value
	//TODO check value length
	b = append(b[:], zo.Value[:]...)

	return b, nil
}

func (zo *zestOptions) Parse(b []byte) error {

	return nil
}

package zest

import "errors"

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

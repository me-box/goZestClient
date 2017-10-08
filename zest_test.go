package zest

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
)

func TestPack4_16_4(t *testing.T) {
	cases := []struct {
		i   uint16
		j   uint16
		k   uint16
		res []byte
		err error
	}{
		{0, 0, 0, []byte{0, 0, 0}, nil},
		{15, 65535, 15, []byte{255, 255, 255}, nil},
		{16, 65535, 15, nil, errors.New("i max is 4 bits")},
		{15, 65535, 16, nil, errors.New("k max is 4 bits")},
	}

	for _, c := range cases {
		b, err := pack4_16_4(c.i, c.j, c.k)
		if bytes.Equal(b, c.res) != true {
			t.Errorf("pack4_16_4 of (%d+%d+%d) was incorrect. \n Expected: %s \n Received: %s",
				c.i,
				c.j,
				c.k,
				hex.Dump(c.res[:]),
				hex.Dump(b[:]),
			)
		}
		if c.err != nil {
			if err.Error() != c.err.Error() {
				t.Errorf("pack4_16_4 of (%d+%d+%d) error was incorrect.  \n Expected: %s \n Received: %s", c.i, c.j, c.k, c.err, err)
			}
		}
	}
}

func TestUnPack4_16_4(t *testing.T) {
	cases := []struct {
		i   uint16
		j   uint16
		k   uint16
		b   []byte
		err error
	}{
		{0, 0, 0, []byte{0, 0, 0}, nil},
		{15, 65535, 15, []byte{255, 255, 255}, nil},
	}

	for _, c := range cases {
		i, j, k := unPack4_16_4(c.b)
		if i != c.i || j != c.j || k != c.k {
			t.Errorf("unPack4_16_4 of (%s) was incorrect. \n Received: (%d,%d%,d)",
				hex.Dump(c.b[:]),
				c.i,
				c.j,
				c.k,
			)
		}
	}
}

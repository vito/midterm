package vt100

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScan(t *testing.T) {
nextCase:
	for _, testCase := range []struct {
		in   string
		want []Command
	}{
		{"fÜ", []Command{
			runeCommand('f'),
			runeCommand('Ü'),
		}},
		{"\u001babc", []Command{
			csiCommand{'a', ""},
			runeCommand('b'),
			runeCommand('c'),
		}},
		{"\u001b[123;31d", []Command{csiCommand{'d', "123;31"}}},
		{"\u009b123;31d", []Command{csiCommand{'d', "123;31"}}},
		{"\u001b123", []Command{
			csiCommand{'1', ""},
			runeCommand('2'),
			runeCommand('3'),
		}},
		{"\u001b[12;\"asd\"s", []Command{
			csiCommand{'s', `12;"asd"`},
		}},
	} {
		s := strings.NewReader(testCase.in)

		for i := 0; i < len(testCase.want); i++ {
			got, unparsed, err := Decode(s)
			if err == io.EOF {
				t.Error("unexpected eof")
				continue nextCase
			}

			assert.Empty(t, unparsed)

			if !assert.Nil(t, err, "unexpected error") {
				continue
			}

			assert.Equal(t, testCase.want[i], got)
		}
		_, unparsed, err := Decode(s)
		assert.Equal(t, err, io.EOF)
		assert.Empty(t, unparsed)
	}
}

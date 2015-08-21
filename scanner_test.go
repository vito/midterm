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
		want []command
	}{
		{"fÜ", []command{
			runeCommand('f'),
			runeCommand('Ü'),
		}},
		{"\u001babc", []command{
			escapeCommand{'a', ""},
			runeCommand('b'),
			runeCommand('c'),
		}},
		{"\u001b[123;31d", []command{escapeCommand{'d', "123;31"}}},
		{"\u009b123;31d", []command{escapeCommand{'d', "123;31"}}},
		{"\u001b123", []command{
			escapeCommand{'1', ""},
			runeCommand('2'),
			runeCommand('3'),
		}},
		{"\u001b[12;\"asd\"s", []command{
			escapeCommand{'s', `12;"asd"`},
		}},
	} {
		s := newScanner(strings.NewReader(testCase.in))

		for i := 0; i < len(testCase.want); i++ {
			got, err := s.next()
			if err == io.EOF {
				t.Error("unexpected eof")
				continue nextCase
			}

			if !assert.Nil(t, err, "unexpected error") {
				continue
			}

			assert.Equal(t, testCase.want[i], got)
		}
		_, err := s.next()
		assert.Equal(t, err, io.EOF)
	}
}

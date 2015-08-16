package vt100

import (
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"testing"
)

func TestScan(t *testing.T) {
nextCase:
	for _, testCase := range []struct {
		in   string
		want []command
	}{
		{"fÜ", []command{
			putRuneCommand('f'),
			putRuneCommand('Ü'),
		}},
		{"\u001babc", []command{
			csCommand{'a', ""},
			putRuneCommand('b'),
			putRuneCommand('c'),
		}},
		{"\u001b[123;31d", []command{csCommand{'d', "123;31"}}},
		{"\u009b123;31d", []command{csCommand{'d', "123;31"}}},
		{"\u001b123", []command{
			csCommand{'1', ""},
			putRuneCommand('2'),
			putRuneCommand('3'),
		}},
		{"\u001b[12;\"asd\"s", []command{
			csCommand{'s', `12;"asd"`},
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

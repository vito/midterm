package vt100

import (
	"io"
	"log"
	"os"
)

var dbg *log.Logger

func init() {
	var logDest io.Writer
	if dest := os.Getenv("VT100_LOGS"); dest != "" {
		logDest, _ = os.Create(dest)
	}
	if logDest == nil {
		logDest = io.Discard
	}
	dbg = log.New(logDest, "", log.LstdFlags)
}

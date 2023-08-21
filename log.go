package midterm

import (
	"io"
	"log"
	"os"
)

var dbg *log.Logger
var trace io.Writer

func init() {
	var logDest io.Writer
	if dest := os.Getenv("VT100_LOGS"); dest != "" {
		logDest, _ = os.Create(dest)
		trace, _ = os.Create(dest + ".raw")
	}
	if logDest == nil {
		logDest = io.Discard
	}

	dbg = log.New(logDest, "", log.LstdFlags)
}

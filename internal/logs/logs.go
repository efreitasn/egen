package logs

import (
	"log"
	"os"
)

// Err is the logger used to print errors.
var Err = log.New(os.Stderr, "err: ", 0)

package fetch

import (
	"os"
)

var (
	removeAll = os.RemoveAll
	mkdir     = os.MkdirAll
)

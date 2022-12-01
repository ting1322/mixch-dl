package inter

import "errors"

var (
	ErrNolive error = errors.New("no live stream")
)
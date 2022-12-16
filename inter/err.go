package inter

import "errors"

var (
	ErrNolive    error = errors.New("no live stream")
	ErrHttpNotOk error = errors.New("http response not OK")
)

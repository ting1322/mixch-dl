package m3u8

import "time"

type ChatDownloader interface {
	SetTime(t time.Duration)
	Count() int
}

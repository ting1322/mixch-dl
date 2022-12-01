package m3u8

import "fmt"

type M3U8 struct {
	version        int
	sequence       int
	targetDuration float64
	tsList         []TsFile
	end            bool
}

type TsFile struct {
	name     string
	duration float64
}

func (ts *TsFile) String() string {
	return fmt.Sprintf("#EXTINF:%v,\r\n%v", ts.duration, ts.name)
}

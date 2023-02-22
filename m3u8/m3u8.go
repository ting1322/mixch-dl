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

func (this *TsFile) String() string {
	return fmt.Sprintf("#EXTINF:%v,\r\n%v", this.duration, this.name)
}

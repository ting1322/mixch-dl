package m3u8

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mixch-dl/inter"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var M3U8FormatError error = errors.New("m3u8 format error")

var PreferFmt string = ""
var FileExt string = ".mp4"

type Downloader struct {
	seq            int
	timer          *time.Timer
	m3u8           *M3U8
	fragCount      int
	totalTime      time.Duration
	Chat           ChatDownloader
	mu             sync.Mutex
	conn           inter.INet
	fs             inter.IFs
	tspartFilename string
	hasMap         bool
	UseInnerAudio  bool
	GuessTs        func(firstTs, baseurl string, downloadedIdx int) []string
	IgnoreTs       func(urlText string) bool // ts line in m3u8
}

func (this *Downloader) GetFragCount() int {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.fragCount
}

func (this *Downloader) incFrag() {
	this.mu.Lock()
	this.fragCount++
	this.mu.Unlock()
}

func (this *Downloader) tryDownloadLostFrag(ctx context.Context, tsw io.Writer, urlList []string) {
	workList := make([]*DownloadWorker, 0)
	for _, urlText := range urlList {
		w := NewWorker(urlText)
		workList = append(workList, w)
		go w.run(ctx, this.conn)
	}
	var successCount int = 0
	var failCount int = 0
	for _, w := range workList {
		result := <-w.complete
		if result.err == nil {
			tsw.Write(result.data)
			this.incFrag()
			successCount++
		} else {
			inter.LogMsg(false, "download missing fragment with error: "+w.url)
			failCount++
		}
	}
	t, err := inter.FfprobeTime(this.tspartFilename)
	if err == nil {
		this.totalTime = t
	}
	inter.LogMsg(false, fmt.Sprintf("REMEDY: success:%v, fail:%v, currentTime:%v\n",
		successCount, failCount, this.totalTime))
}

func (this *Downloader) tryAppendBaseUrl(baseUrl string, suffix_url string) string {
	var url string
	if strings.HasPrefix(suffix_url, "../") {
		url = baseUrl[0:strings.LastIndex(baseUrl, "/")] + "/" + suffix_url[3:]
	} else if strings.HasPrefix(suffix_url, "http") {
		url = suffix_url
	} else {
		url = baseUrl + "/" + suffix_url
	}
	return url
}

func (this *Downloader) tryDownloadXMap(ctx context.Context, tsw io.Writer, url string) error {
	data, err := this.conn.GetFile(ctx, url)
	if err != nil {
		return fmt.Errorf("err: %w, url: %v", err, url)
	}
	tsw.Write(data)
	return nil
}

func (this *Downloader) downloadAndWrite(ctx context.Context, m3u8Url string, tsw io.Writer) error {
	m3u8, err := this.downloadM3U8(ctx, m3u8Url, this.conn)
	this.m3u8 = m3u8
	if err != nil {
		return err
	}

	baseurl := m3u8Url[0:strings.LastIndex(m3u8Url, "/")]

	if m3u8.x_map != "" && !this.hasMap {
		url := this.tryAppendBaseUrl(baseurl, m3u8.x_map)
		err = this.tryDownloadXMap(ctx, tsw, url)
		if err != nil {
			return err
		}
		this.hasMap = true
	}

	if m3u8.sequence > (this.seq+1) && len(m3u8.tsList) > 0 && this.GuessTs != nil {
		urlList := this.GuessTs(this.m3u8.tsList[0].name, baseurl, this.seq)
		if urlList != nil && len(urlList) > 0 {
			this.tryDownloadLostFrag(ctx, tsw, urlList)
		}
	}

	for idx, ts := range m3u8.tsList {
		select {
		case <-ctx.Done():
			return errors.New("Cancel")
		default:
			seq := m3u8.sequence + idx
			if seq <= this.seq {
				if seq < this.seq-15 {
					inter.LogMsg(true, fmt.Sprintf("m3u8 sequence reset, orig:%v, current:%v", this.seq, seq))
				} else {
					continue
				}
			}
			url := this.tryAppendBaseUrl(baseurl, ts.name)
			data, err := this.conn.GetFile(ctx, url)
			if err != nil {
				return fmt.Errorf("err: %w, url: %v", err, url)
			}
			tsw.Write(data)
			this.totalTime += time.Duration(ts.duration * float64(time.Second))
			if (this.seq+1) != seq && this.seq != 0 {
				inter.LogMsg(true, "some video fragnment missing, re-sync chat log")
				this.Chat.SetTime(this.totalTime)
			}
			this.seq = seq
			this.incFrag()
		}
	}
	return nil
}

func (this *Downloader) downloadMergeLoop(ctx context.Context, m3u8Url string) {
	tspart, err := this.fs.Create(this.tspartFilename)
	if err != nil {
		log.Println("m3u8/downloadMergeLoop", err)
		return
	}
	defer tspart.Close()

	retry := 30
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := this.downloadAndWrite(ctx, m3u8Url, tspart)
			var nestedErr NestedM3u8Error
			if err == nil {
				retry = 30
				this.timer.Reset(1500 * time.Millisecond)
				//inter.LogProgress(this.GetFragCount(), this.Chat.Count(), this.totalTime)
			} else if errors.Is(err, M3U8FormatError) || errors.Is(err, inter.ErrHttpNotOk) {
				log.Printf("stream end? %v\n", err)
				retry = 0
			} else if errors.As(err, &nestedErr) {
				queryIdx := strings.Index(m3u8Url, "?")
				baseurl := m3u8Url
				if queryIdx > 0 {
					baseurl = baseurl[0:queryIdx]
				}
				baseurl = baseurl[0:strings.LastIndex(baseurl, "/")]
				m3u8Url = this.tryAppendBaseUrl(baseurl, nestedErr.url)
				log.Println("sub m3u8: ", m3u8Url)
				retry--
			} else if ctx.Err() == context.Canceled {
				retry = 0
			} else {
				log.Printf("download with error: %v, retry %v\n", err, retry)
				this.timer.Reset(1000 * time.Millisecond)
				retry--
			}
			if (this.m3u8 != nil && this.m3u8.end) || retry <= 0 {
				this.timer.Stop()
				inter.LogStatus(inter.STATUS_Finish)
				return
			}
			<-this.timer.C
		}
	}
}

func (this *Downloader) DownloadPart(ctx context.Context, m3u8Url string, conn inter.INet, fs inter.IFs, filename string) {
	this.timer = time.NewTimer(2 * time.Second)
	this.conn = conn
	this.fs = fs
	this.tspartFilename = filename
	this.downloadMergeLoop(ctx, m3u8Url)
	if fs.Exist(this.tspartFilename) {
		if this.GetFragCount() == 0 {
			fs.Delete(this.tspartFilename)
		}
	}
}

func (this *Downloader) DownloadMerge(ctx context.Context, m3u8Url string, conn inter.INet, fs inter.IFs, filename string) {
	this.timer = time.NewTimer(2 * time.Second)
	this.conn = conn
	this.fs = fs
	this.tspartFilename = filename + ".ts.part"
	this.downloadMergeLoop(ctx, m3u8Url)
	if fs.Exist(this.tspartFilename) {
		if this.GetFragCount() == 0 {
			fs.Delete(this.tspartFilename)
		} else {
			inter.FfmpegMerge(this.tspartFilename, filename+FileExt, false)
		}
	}
}

type ResolutionM3u8 struct {
	resolution string
	url        string
}

func (this *Downloader) downloadM3U8(ctx context.Context, url string, conn inter.INet) (*M3U8, error) {
	text, err := conn.GetWebPage(ctx, url)
	if err != nil {
		return nil, err
	}
	if len(text) == 0 {
		inter.LogMsg(true, "m3u8 empty")
		return nil, M3U8FormatError
	}
	var time float64
	m3u8 := &M3U8{}

	tsRe, _ := regexp.Compile(`\.(ts|m4v|m4a)`)
	tsNeedSave := func(line string) bool {
		isTs := tsRe.FindStringSubmatch(line)
		ignore := this.IgnoreTs != nil && this.IgnoreTs(line)
		return isTs != nil && !ignore
	}

	reExtinf, _ := regexp.Compile(`#EXTINF:(\d+(\.\d+)?)`)
	for _, line := range strings.Split(text, "\n") {
		if len(line) > 0 && line[0] == '#' {
			match := reExtinf.FindStringSubmatch(line)
			if match != nil || len(match) == 1 {
				time, err = strconv.ParseFloat(match[1], 64)
				if err != nil {
					return nil, err
				}
			} else if strings.HasPrefix(line, "#EXT-X-VERSION:") {
				m3u8.version, _ = strconv.Atoi(strings.TrimPrefix(line, "#EXT-X-VERSION:"))
			} else if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
				m3u8.sequence, _ = strconv.Atoi(strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"))
			} else if strings.HasPrefix(line, "#EXT-X-TARGETDURATION:") {
				m3u8.targetDuration, _ = strconv.ParseFloat(strings.TrimPrefix(line, "#EXT-X-TARGETDURATION:"), 64)
			} else if strings.HasPrefix(line, "#EXT-X-MAP:URI=") {
				uri := strings.TrimPrefix(line, "#EXT-X-MAP:URI=")
				if strings.HasPrefix(uri, "\"") {
					uri = uri[1 : len(uri)-2]
				}
				m3u8.x_map = uri
			} else if strings.HasPrefix(line, "#EXT-X-ENDLIST") {
				m3u8.end = true
			}
			continue
		} else if tsNeedSave(line) {
			ts := TsFile{}
			ts.duration = time
			ts.name = line
			m3u8.tsList = append(m3u8.tsList, ts)
		}
	}
	if m3u8.version == 0 || len(m3u8.tsList) == 0 {
		if strings.Contains(text, "#EXT-X-MEDIA:TYPE=AUDIO") && this.UseInnerAudio {
			re, _ := regexp.Compile(`#EXT-X-MEDIA:TYPE=AUDIO.+,URI="(.+\.m3u8)"`)
			match := re.FindStringSubmatch(text)
			if match != nil {
				return nil, NestedM3u8Error{url: match[1]}
			}
		}
		if strings.Contains(text, "#EXT-X-STREAM-INF:") {
			subM3u8 := make(map[string]string, 0)
			cur_resolution := ""
			re, _ := regexp.Compile(`(VIDEO|RESOLUTION)="?(\w+)"?`)
			for _, line := range strings.Split(text, "\n") {
				if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
					match := re.FindStringSubmatch(line)
					if match == nil || len(match) < 1 {
						cur_resolution = ""
					} else {
						cur_resolution = match[2]
					}
				} else if !strings.HasPrefix(line, "#") {
					subM3u8[cur_resolution] = line
				}
			}
			if len(subM3u8) > 0 {
				if PreferFmt != "" {
					if u, exist := subM3u8[PreferFmt]; exist {
						return nil, NestedM3u8Error{url: u}
					}
				}
				if u, exist := subM3u8["1920x1080"]; exist {
					return nil, NestedM3u8Error{url: u}
				}
				if u, exist := subM3u8["1280x720"]; exist {
					return nil, NestedM3u8Error{url: u}
				}
				if u, exist := subM3u8["720p60"]; exist {
					return nil, NestedM3u8Error{url: u}
				}
				if u, exist := subM3u8["720p30"]; exist {
					return nil, NestedM3u8Error{url: u}
				}
				if u, exist := subM3u8["480p30"]; exist {
					return nil, NestedM3u8Error{url: u}
				}
				if u, exist := subM3u8["360p30"]; exist {
					return nil, NestedM3u8Error{url: u}
				}
				if u, exist := subM3u8["160p30"]; exist {
					return nil, NestedM3u8Error{url: u}
				}
				if u, exist := subM3u8["audio_only"]; exist {
					return nil, NestedM3u8Error{url: u}
				}
			}
		}

		inter.LogMsg(true, "m3u8 parse error: "+text)
		return nil, M3U8FormatError
	}
	return m3u8, nil
}

type NestedM3u8Error struct {
	url string
}

func (m NestedM3u8Error) Error() string {
	return "it is a nested m3u8, use new url download again. " + m.url
}

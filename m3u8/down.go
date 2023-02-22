package m3u8

import (
	"context"
	"errors"
	"fmt"
	"inter"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

var M3U8FormatError error = errors.New("m3u8 format error")

type Downloader struct {
	seq            int
	timer          *time.Timer
	m3u8           *M3U8
	fragCount      int64
	totalTime      time.Duration
	Chat           ChatDownloader
	mu             sync.Mutex
	conn           inter.INet
	fs             inter.IFs
	tspartFilename string
	GuessTs        func(firstTs, baseurl string, downloadedIdx int) []string
	IgnoreTs       func(urlText string) bool // ts line in m3u8
}

func (this *Downloader) GetFragCount() int64 {
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
			log.Println("frag fail:", w.url)
			failCount++
		}
	}
	t, err := inter.FfprobeTime(this.tspartFilename)
	if err == nil {
		this.totalTime = t
	}
	log.Printf("REMEDY: success:%v, fail:%v, currentTime:%v\n", successCount, failCount, this.totalTime)
}

func (this *Downloader) downloadAndWrite(ctx context.Context, m3u8Url string, tsw io.Writer) error {
	m3u8, err := this.downloadM3U8(ctx, m3u8Url, this.conn)
	this.m3u8 = m3u8
	if err != nil {
		return err
	}

	baseurl := m3u8Url[0:strings.LastIndex(m3u8Url, "/")]

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
					log.Printf("m3u8 sequence reset, orig:%v, current:%v", this.seq, seq)
				} else {
					continue
				}
			}
			var url string
			if strings.HasPrefix(ts.name, "../") {
				url = baseurl[0:strings.LastIndex(baseurl, "/")] + "/" + ts.name[3:]
			} else {
				url = baseurl + "/" + ts.name
			}
			data, err := this.conn.GetFile(ctx, url)
			if err != nil {
				return fmt.Errorf("err: %w, url: %v", err, url)
			}
			tsw.Write(data)
			this.totalTime += time.Duration(ts.duration * float64(time.Second))
			if (this.seq+1) != seq && this.seq != 0 {
				log.Printf("some video fragnment missing, re-sync chat log")
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
			if err == nil {
				retry = 30
				this.timer.Reset(1500 * time.Millisecond)

				fmt.Printf("downloaded video fragment: %d, duration: %v, chat: %v\n", this.GetFragCount(),
					this.totalTime, this.Chat.Count())
			} else if errors.Is(err, M3U8FormatError) || errors.Is(err, inter.ErrHttpNotOk) {
				log.Printf("stream end? %v\n", err)
				retry = 0
			} else if ctx.Err() == context.Canceled {
				retry = 0
			} else {
				log.Printf("download with error: %v, retry %v\n", err, retry)
				this.timer.Reset(1000 * time.Millisecond)
				retry--
			}
			if (this.m3u8 != nil && this.m3u8.end) || retry <= 0 {
				this.timer.Stop()
				fmt.Println()
				log.Println("download finish")
				return
			}
			<-this.timer.C
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
			inter.FfmpegMerge(this.tspartFilename, filename+".mp4", false)
		}
	}
}

func (this *Downloader) downloadM3U8(ctx context.Context, url string, conn inter.INet) (*M3U8, error) {
	text, err := conn.GetWebPage(ctx, url)
	if err != nil {
		return nil, err
	}
	if len(text) == 0 {
		return nil, M3U8FormatError
	}
	var time float64
	m3u8 := &M3U8{}
	for _, line := range strings.Split(text, "\n") {
		if len(line) > 0 && line[0] == '#' {
			if strings.HasPrefix(line, "#EXTINF:") {
				timeText := line[8 : len(line)-1]
				time, err = strconv.ParseFloat(timeText, 64)
				if err != nil {
					return nil, err
				}
			} else if strings.HasPrefix(line, "#EXT-X-VERSION:") {
				m3u8.version, _ = strconv.Atoi(strings.TrimPrefix(line, "#EXT-X-VERSION:"))
			} else if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
				m3u8.sequence, _ = strconv.Atoi(strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"))
			} else if strings.HasPrefix(line, "#EXT-X-TARGETDURATION:") {
				m3u8.targetDuration, _ = strconv.ParseFloat(strings.TrimPrefix(line, "#EXT-X-TARGETDURATION:"), 64)
			} else if strings.HasPrefix(line, "#EXT-X-ENDLIST") {
				m3u8.end = true
			}
			continue
		} else if strings.HasSuffix(line, ".ts") && !this.IgnoreTs(line) {
			ts := TsFile{}
			ts.duration = time
			ts.name = line
			m3u8.tsList = append(m3u8.tsList, ts)
		}
	}
	if m3u8.version == 0 || len(m3u8.tsList) == 0 {
		return nil, M3U8FormatError
	}
	return m3u8, nil
}

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
	downloadedSeq int
	timer         *time.Timer
	m3u8          *M3U8
	fragCount     int64
	mu            sync.Mutex
}

func (d *Downloader) GetFragCount() int64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.fragCount
}

func (d *Downloader) incFrag() {
	d.mu.Lock()
	d.fragCount++
	d.mu.Unlock()
}

func (d *Downloader) downloadAndWrite(ctx context.Context, m3u8Url string, conn inter.INet, tsw io.Writer) error {
	m3u8, err := downloadM3U8(ctx, m3u8Url, conn)
	d.m3u8 = m3u8
	if err != nil {
		return err
	}

	baseurl := m3u8Url[0:strings.LastIndex(m3u8Url, "/")]

	for idx, ts := range m3u8.tsList {
		select {
		case <-ctx.Done():
			return errors.New("Cancel")
		default:
			seq := m3u8.sequence + idx
			if seq <= d.downloadedSeq {
				continue
			}
			url := baseurl + "/" + ts.name
			data, err := conn.GetFile(ctx, url)
			if err != nil {
				return err
			}
			tsw.Write(data)
			d.downloadedSeq = seq
			d.incFrag()
		}
	}
	return nil
}

func (d *Downloader) downloadMergeLoop(ctx context.Context, m3u8Url string, conn inter.INet, fs inter.IFs, tspartFilename string) {
	tspart, err := fs.Create(tspartFilename)
	if err != nil {
		log.Println("m3u8/downloadMergeLoop", err)
		return
	}
	defer tspart.Close()

	retry := 20
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := d.downloadAndWrite(ctx, m3u8Url, conn, tspart)
			if err == nil {
				retry = 20
			} else if err == M3U8FormatError {
				fmt.Println()
				log.Printf("\nm3u8 format error, stream end?")
				retry = 0
			} else if ctx.Err() == context.Canceled {
				retry = 0
			} else {
				fmt.Println()
				log.Printf("\ndownload with error: %v, retry %v\n", err, retry)
				retry--
			}
			if (d.m3u8 != nil && d.m3u8.end) || retry <= 0 {
				d.timer.Stop()
				fmt.Println()
				log.Println("\ndownload finish")
				return
			}
			d.timer.Reset(1500 * time.Millisecond)
			<-d.timer.C
			fmt.Printf("\rdownloaded video fragment: %d   ", d.GetFragCount())
		}
	}
}

func (d *Downloader) DownloadMerge(ctx context.Context, m3u8Url string, conn inter.INet, fs inter.IFs, filename string) {
	d.timer = time.NewTimer(2 * time.Second)
	tspartFilename := filename + ".ts.part"
	d.downloadMergeLoop(ctx, m3u8Url, conn, fs, tspartFilename)
	if fs.Exist(tspartFilename) {
		inter.FfmpegMerge(tspartFilename, filename+".mp4", false)
	}
}

func downloadM3U8(ctx context.Context, url string, conn inter.INet) (*M3U8, error) {
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
					log.Fatal(err)
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
		} else if strings.HasSuffix(line, ".ts") {
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

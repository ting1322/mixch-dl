package chzzk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mixch-dl/inter"
	"mixch-dl/m3u8"
	"strings"
	"time"

	"github.com/ting1322/chat-player/pkg/cplayer"
)

var DownloadChatRoom bool = true

type Chzzk struct {
	Id           string
	Name         string
	liveId       string
	M3u8Url      string
	imgUrl       string
	title        string
	Chat         string
	status       string
	vd           *m3u8.Downloader
}

func New(text string) *Chzzk {
	// example: https://www.spooncast.net/jp/live/@lala_ukulele
	if strings.HasPrefix(text, "https://chzzk.naver.com/live/") {
		text = strings.TrimPrefix(text, "https://chzzk.naver.com/live/")
		idx := strings.Index(text, "/")
		if idx > 0 {
			text = text[0:idx]
		}
		return &Chzzk{Id: text}
	}
	return nil
}

func (this *Chzzk) WaitStreamStart(ctx context.Context, conn inter.INet) error {
	err := this.LoadLiveDetail(ctx, conn)
	if errors.Is(err, inter.ErrNolive) {
		log.Println("wait stream start......")
		err = this.waitLiveLoop(ctx, 30*time.Second, conn)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (this *Chzzk) waitLiveLoop(ctx context.Context, interval time.Duration, conn inter.INet) error {
	timer := time.NewTimer(interval)
	for {
		<-timer.C
		err := this.LoadLiveDetail(ctx, conn)
		if err == nil {
			log.Println("live start.")
			return nil
		}
		if !errors.Is(err, inter.ErrNolive) {
			return fmt.Errorf("wait live start: %w", err)
		}
		timer.Reset(interval)
	}
}

func (this *Chzzk) LoadLiveDetail(ctx context.Context, conn inter.INet) error {
	url := fmt.Sprintf("https://api.chzzk.naver.com/service/v2/channels/%v/live-detail", this.Id)
	webText, err := conn.GetWebPage(ctx, url)
	if err != nil {
		return err
	}

	if !this.parseLiveDetail(webText) {
		return inter.ErrNolive
	}

	if this.status != "OPEN" {
		return inter.ErrNolive
	}

	log.Println("m3u8 url:", this.M3u8Url)

	return nil
}

type jmap = map[string]any

func (this *Chzzk) parseLiveDetail(jsonText string) bool {
	var jsonmap jmap
	decoder := json.NewDecoder(strings.NewReader(jsonText))
	decoder.UseNumber()
	decoder.Decode(&jsonmap)
	//json.Unmarshal([]byte(jsonText), &jsonmap)
	if content, exist := jsonmap["content"]; exist {
		if status, exist := content.(jmap)["status"]; exist {
			this.status = status.(string)
		}
		if channel, exist := content.(jmap)["channel"]; exist {
			if channelName, exist := channel.(jmap)["channelName"]; exist {
				this.Name = channelName.(string)
			}
		}
		if liveTitle, exist := content.(jmap)["liveTitle"]; exist {
			this.title = liveTitle.(string)
		}
		//if liveImageUrl, exist := content.(jmap)["liveImageUrl"]; exist {
		//	this.imgUrl = liveImageUrl.(string)
		//}
		if livePlaybackJson, exist := content.(jmap)["livePlaybackJson"]; exist {
			var jsonmap2 jmap
			decoder2 := json.NewDecoder(strings.NewReader(livePlaybackJson.(string)))
			decoder2.UseNumber()
			decoder2.Decode(&jsonmap2)
			for _, media := range jsonmap2["media"].([]any) {
				if path, exist := media.(jmap)["path"]; exist {
					this.M3u8Url = path.(string)
					break
				}
			}
			if meta, exist := jsonmap2["meta"]; exist {
				if videoId, exist := meta.(jmap)["videoId"]; exist {
					this.liveId = videoId.(string)
				}
			}
		}
	}
	return this.M3u8Url != ""
}

func (this *Chzzk) Download(ctx context.Context, netconn inter.INet, fio inter.IFs, filename string) error {
	//ctx2, cancel := context.WithCancel(ctx)

	coverCh := make(chan string, 1)
	if this.imgUrl == "" {
		coverCh <- ""
	} else {
		coverFileName, err := inter.DownloadThumbnail(ctx, netconn, fio, filename, this.imgUrl)
		if err != nil {
			coverCh <- ""
		} else {
			coverCh <- coverFileName
		}
	}

	this.vd = &m3u8.Downloader{}
	this.vd.DownloadMerge(ctx, this.M3u8Url, netconn, fio, filename)
	//cancel()
	if this.vd.GetFragCount() == 0 {
		return inter.ErrNolive
	}

	videoCount, err := inter.FfprobeVideoCount(filename + ".mp4")
	if err != nil {
		inter.LogMsg(false, "error: cannot get video stream count from mp4 file")
		videoCount = 1
	}
	coverFile := <-coverCh
	if coverFile != "" {
		inter.FfmpegAttachThumbnail(filename+".mp4", coverFile, videoCount)
	}
	if this.title != "" {
		meta := inter.FfmpegMeta{
			Title:  this.title,
			Artist: this.Name,
			Album:  fmt.Sprintf("%v-%v", this.Name, this.title),
		}
		inter.FfmpegMetadata(filename+".mp4", meta)
	}
	inter.FfmpegFastStartMp4(filename + ".mp4")
	return nil
}

func generateHtml(mp4 string) {
	option := cplayer.NewOption()
	cplayer.ProcessVideo(option, mp4)
}

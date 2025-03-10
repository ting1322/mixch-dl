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
	Id       string
	Name     string
	liveId   string
	M3u8Url  string
	M3u8UrlA string
	imgUrl   string
	title    string
	Chat     string
	status   string
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

	log.Println("video m3u8 url:", this.M3u8Url)
	log.Println("audio m3u8 url:", this.M3u8UrlA)

	return nil
}

type jmap = map[string]any

func (this *Chzzk) parseLiveDetail(jsonText string) bool {
	var jsonmap jmap
	decoder := json.NewDecoder(strings.NewReader(jsonText))
	decoder.UseNumber()
	decoder.Decode(&jsonmap)
	//json.Unmarshal([]byte(jsonText), &jsonmap)
	content, exist := jsonmap["content"]
	if !exist {
		return false
	}
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
	livePlaybackJson, exist := content.(jmap)["livePlaybackJson"]
	if !exist {
		return false
	}
	if !this.parseLivePlayback(livePlaybackJson.(string)) {
		return false
	}
	return true
}

func (this *Chzzk) parseLivePlayback(jsonText string) bool {
	var jsonmap jmap
	decoder := json.NewDecoder(strings.NewReader(jsonText))
	decoder.UseNumber()
	decoder.Decode(&jsonmap)
	var hls_media jmap
	for _, media := range jsonmap["media"].([]any) {
		if mediaId, exist := media.(jmap)["mediaId"]; exist {
			if mediaId.(string) == "HLS" {
				hls_media = media.(jmap)
				break
			}
		}
	}
	if hls_media == nil {
		return false
	}
	if path, exist := hls_media["path"]; exist {
		this.M3u8Url = path.(string)
	}
	for _, encodingTrack := range hls_media["encodingTrack"].([]any) {
		encodingTrackId, exist := encodingTrack.(jmap)["encodingTrackId"]
		if !exist {
			continue
		}
		if encodingTrackId.(string) != "alow.stream" {
			continue
		}
		path, exist := encodingTrack.(jmap)["path"]
		if !exist {
			continue
		}
		this.M3u8UrlA = path.(string)
	}

	if meta, exist := jsonmap["meta"]; exist {
		if videoId, exist := meta.(jmap)["videoId"]; exist {
			this.liveId = videoId.(string)
		}
	}
	return this.M3u8Url != "" && this.M3u8UrlA != ""
}

func (this *Chzzk) Download(ctx context.Context, netconn inter.INet, fio inter.IFs, filename string) error {
	//ctx2, cancel := context.WithCancel(ctx)

	vch := make(chan string, 1)
	ach := make(chan string, 1)
	go func() {
		vd := &m3u8.Downloader{}
		vfilename := filename + "-video.part"
		vd.DownloadPart(ctx, this.M3u8Url, netconn, fio, vfilename)
		if vd.GetFragCount() == 0 {
			vch <- ""
		}
		vch <- vfilename
	}()
	go func() {
		ad := &m3u8.Downloader{UseInnerAudio: true}
		afilename := filename + "-audio.part"
		ad.DownloadPart(ctx, this.M3u8Url, netconn, fio, afilename)
		if ad.GetFragCount() == 0 {
			ach <- ""
		}
		ach <- afilename
	}()

	vfilename := <-vch
	afilename := <-ach

	inter.FfmpegMergeAV(vfilename, afilename, filename+m3u8.FileExt, true)

	if this.title != "" {
		meta := inter.FfmpegMeta{
			Title:  this.title,
			Artist: this.Name,
			Album:  fmt.Sprintf("%v-%v", this.Name, this.title),
		}
		inter.FfmpegMetadata(filename+m3u8.FileExt, meta)
	}
	inter.FfmpegFastStartMp4(filename + m3u8.FileExt)
	return nil
}

func generateHtml(mp4 string) {
	option := cplayer.NewOption()
	cplayer.ProcessVideo(option, mp4)
}

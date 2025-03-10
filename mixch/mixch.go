package mixch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mixch-dl/inter"
	"mixch-dl/m3u8"
	"regexp"
	"strings"
	"time"

	"github.com/ting1322/chat-player/pkg/cplayer"
)

type Mixch struct {
	imgUrl  string
	Name    string
	Id      string
	M3u8Url string
	Chat    string
	vd      *m3u8.Downloader
}

func New(text string) (*Mixch, error) {
	if strings.HasPrefix(text, "https://mixch.tv/u/") {
		text = strings.TrimPrefix(text, "https://mixch.tv/u/")
		idx := strings.Index(text, "/")
		if idx > 0 {
			text = text[0:idx]
		}
		return &Mixch{Id: text}, nil
	}
	re, _ := regexp.Compile(`^\d+$`)
	if re.MatchString(text) {
		return &Mixch{Id: text}, nil
	}
	return nil, errors.New("unknown mixch id")
}

func (this *Mixch) WaitStreamStart(ctx context.Context, conn inter.INet) error {
	err := this.LoadUserPage(ctx, conn)
	if errors.Is(err, inter.ErrNolive) {
		inter.LogMsg(false, "wait stream start......")
		err = this.waitLiveLoop(ctx, 15*time.Second, conn)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (this *Mixch) waitLiveLoop(ctx context.Context, interval time.Duration, conn inter.INet) error {
	timer := time.NewTimer(interval)
	for {
		<-timer.C
		err := this.LoadUserPage(ctx, conn)
		if err == nil {
			inter.LogMsg(false, "live start.")
			return nil
		}
		if !errors.Is(err, inter.ErrNolive) {
			return fmt.Errorf("wait live start: %w", err)
		}
		timer.Reset(interval)
	}
}

func (this *Mixch) LoadUserPage(ctx context.Context, conn inter.INet) error {
	url := fmt.Sprintf("https://mixch.tv/u/%v/live", this.Id)
	webText, err := conn.GetWebPage(ctx, url)
	if err != nil {
		return err
	}

	if !this.parseLivePage(webText) || len(this.M3u8Url) == 0 {
		return inter.ErrNolive
	}

	inter.LogMsg(false, fmt.Sprint("m3u8 url:", this.M3u8Url))

	return nil
}

func (this *Mixch) Download(ctx context.Context, netconn inter.INet, fio inter.IFs, filename string) error {
	ctx2, cancel := context.WithCancel(ctx)
	chat := &Chat{Fs: fio}
	var cs chan int
	if len(this.Chat) > 0 {
		cs = make(chan int, 1)
		go func() {
			chat.Connect(ctx2, this.Chat, filename)
			cs <- 1
		}()
	}

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

	this.vd = &m3u8.Downloader{
		Chat:    chat,
		GuessTs: guessTs,
	}
	this.vd.DownloadMerge(ctx, this.M3u8Url, netconn, fio, filename)
	cancel()
	if cs != nil {
		<-cs
	}
	if this.vd.GetFragCount() == 0 {
		return inter.ErrNolive
	}

	coverFile := <-coverCh
	if coverFile != "" {
		inter.FfmpegAttachThumbnail(filename+m3u8.FileExt, coverFile, 2)
	}
	if this.Name != "" {
		meta := inter.FfmpegMeta{Artist: this.Name, Album: this.Name}
		inter.FfmpegMetadata(filename+m3u8.FileExt, meta)
	}

	inter.FfmpegFastStartMp4(filename + m3u8.FileExt)
	generateHtml(filename + m3u8.FileExt)
	return nil
}

func generateHtml(mp4 string) {
	option := cplayer.NewOption()
	cplayer.ProcessVideo(option, mp4)
}

func (this *Mixch) parseLivePage(htmContent string) bool {
	prefix := "window.__INITIAL_JS_STATE__ = "
	for _, line := range strings.Split(htmContent, "\n") {
		if strings.HasPrefix(line, prefix) {
			text := strings.TrimPrefix(line, prefix)
			text = strings.TrimSuffix(text, ";")
			return this.parseLivePageJson(text)
		}
	}
	return false
}

type jmap = map[string]any

func (this *Mixch) parseLivePageJson(jsonText string) bool {
	var jsonmap jmap
	json.Unmarshal([]byte(jsonText), &jsonmap)
	liveInfo, exist := jsonmap["liveInfo"].(jmap)
	if exist {
		hls, exist := liveInfo["hls"]
		if !exist {
			return false
		}
		this.M3u8Url = hls.(string)
		chat, exist := liveInfo["chat"]
		if exist {
			this.Chat = chat.(string)
		}
	}
	broInfo, exist := jsonmap["broadcasterInfo"].(jmap)
	if exist {
		name, exist := broInfo["name"].(string)
		if exist {
			this.Name = name
		}
		imageUrl, exist := broInfo["profile_image_url"].(string)
		if exist {
			this.imgUrl = imageUrl
		}
	}

	return true
}

package spoon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ting1322/mixch-dl/inter"
	"github.com/ting1322/mixch-dl/m3u8"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/ting1322/chat-player/pkg/cplayer"
)

type Spoon struct {
	Id           string
	Name         string
	liveId       string
	M3u8Url      string
	imgUrl       string
	title        string
	Chat         string
	jsAppVersion string
	vd           *m3u8.Downloader
}

func New(text string) *Spoon {
	// example: https://www.spooncast.net/jp/live/@lala_ukulele
	if strings.HasPrefix(text, "https://www.spooncast.net/jp/live/@") {
		text = strings.TrimPrefix(text, "https://www.spooncast.net/jp/live/@")
		idx := strings.Index(text, "/")
		if idx > 0 {
			text = text[0:idx]
		}
		return &Spoon{Id: text}
	}
	return nil
}

func (this *Spoon) WaitStreamStart(ctx context.Context, conn inter.INet) error {
	err := this.LoadUserPage(ctx, conn)
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

func (this *Spoon) waitLiveLoop(ctx context.Context, interval time.Duration, conn inter.INet) error {
	timer := time.NewTimer(interval)
	for {
		<-timer.C
		err := this.LoadUserPage(ctx, conn)
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

func (this *Spoon) LoadUserPage(ctx context.Context, conn inter.INet) error {
	url := fmt.Sprintf("https://jp-api.spooncast.net/profiles/%v/meta/", this.Id)
	webText, err := conn.GetWebPage(ctx, url)
	if err != nil {
		return err
	}

	if !this.parseMetaPage(webText) {
		return inter.ErrNolive
	}

	// https://jp-api.spooncast.net/lives/35034345/
	url = fmt.Sprintf("https://jp-api.spooncast.net/lives/%v/", this.liveId)
	webText, err = conn.GetWebPage(ctx, url)
	if err != nil {
		return err
	}
	if !this.parseLiveInfoPage(webText) {
		return inter.ErrNolive
	}

	err = this.LoadJsVersion(ctx, conn)
	if err != nil {
		log.Println("find js appversion fail, use default value,", err)
		this.jsAppVersion = "8.0.1"
	}

	log.Println("m3u8 url:", this.M3u8Url)
	log.Println("js appversion:", this.jsAppVersion)

	return nil
}

func (this *Spoon) LoadJsVersion(ctx context.Context, conn inter.INet) error {
	url := fmt.Sprintf(`https://www.spooncast.net/jp/live/@%v`, this.Id)
	webText, err := conn.GetWebPage(ctx, url)
	if err != nil {
		return err
	}
	re, _ := regexp.Compile(`script src="/(src/js/main\.\w+\.chunk\.js)"`)
	match := re.FindStringSubmatch(webText)
	if match == nil || len(match) < 1 {
		return errors.New("not found main.*.js")
	}
	jsurl := `https://www.spooncast.net/` + match[1]

	webText, err = conn.GetWebPage(ctx, jsurl)

	// find: appVersion:"8.0.1"
	re, _ = regexp.Compile(`appVersion:"([0-9\.]+)"`)
	match = re.FindStringSubmatch(webText)
	if match == nil || len(match) < 1 {
		return errors.New("not found appversion in main.*.js")
	}
	this.jsAppVersion = match[1]
	return nil
}

type jmap = map[string]any

func (this *Spoon) parseMetaPage(jsonText string) bool {
	var jsonmap jmap
	decoder := json.NewDecoder(strings.NewReader(jsonText))
	decoder.UseNumber()
	decoder.Decode(&jsonmap)
	//json.Unmarshal([]byte(jsonText), &jsonmap)
	for _, result := range jsonmap["results"].([]any) {
		if liveId, exist := result.(jmap)["current_live_id"]; exist {
			if liveId != nil { // nil if offline
				this.liveId = fmt.Sprint(liveId)
				this.Chat = fmt.Sprintf(`wss://jp-heimdallr.spooncast.net/%v`, liveId)
			}
		}
	}
	return this.liveId != ""
}

func (this *Spoon) parseLiveInfoPage(jsonText string) bool {
	var jsonmap jmap
	json.Unmarshal([]byte(jsonText), &jsonmap)
	for _, result := range jsonmap["results"].([]any) {
		if url_hls, exist := result.(jmap)["url_hls"]; exist {
			this.M3u8Url = url_hls.(string)
		}
		if img_url, exist := result.(jmap)["img_url"]; exist {
			this.imgUrl = img_url.(string)
		}
		if title, exist := result.(jmap)["title"]; exist {
			this.title = title.(string)
		}
		if author, exist := result.(jmap)["author"]; exist {
			if nickname, exist := author.(jmap)["nickname"]; exist {
				this.Name = nickname.(string)
			}
		}
	}
	return this.M3u8Url != ""
}

func (this *Spoon) Download(ctx context.Context, netconn inter.INet, fio inter.IFs, filename string) error {
	ctx2, cancel := context.WithCancel(ctx)
	chat := &Chat{
		Fs:           fio,
		liveId:       this.liveId,
		jsAppVersion: this.jsAppVersion,
	}
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

	ignorePattern := func(urlText string) bool {
		if strings.Contains(urlText, "/prerole/media") {
			return true
		}
		return false
	}
	this.vd = &m3u8.Downloader{
		Chat:     chat,
		GuessTs:  guessTs,
		IgnoreTs: ignorePattern,
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
		inter.FfmpegAttachThumbnail(filename+".mp4", coverFile, 1)
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
	generateHtml(filename + ".mp4")
	return nil
}

func generateHtml(mp4 string) {
	option := cplayer.NewOption()
	cplayer.ProcessVideo(option, mp4)
}

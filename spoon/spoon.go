package spoon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"inter"
	"log"
	"m3u8"
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

func (m *Spoon) WaitStreamStart(ctx context.Context, conn inter.INet) error {
	err := m.LoadUserPage(ctx, conn)
	if errors.Is(err, inter.ErrNolive) {
		log.Println("wait stream start......")
		err = m.waitLiveLoop(ctx, 30*time.Second, conn)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (m *Spoon) waitLiveLoop(ctx context.Context, interval time.Duration, conn inter.INet) error {
	timer := time.NewTimer(interval)
	for {
		<-timer.C
		err := m.LoadUserPage(ctx, conn)
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

func (m *Spoon) LoadUserPage(ctx context.Context, conn inter.INet) error {
	url := fmt.Sprintf("https://jp-api.spooncast.net/profiles/%v/meta/", m.Id)
	webText, err := conn.GetWebPage(ctx, url)
	if err != nil {
		return err
	}

	if !m.parseMetaPage(webText) {
		return inter.ErrNolive
	}

	// https://jp-api.spooncast.net/lives/35034345/
	url = fmt.Sprintf("https://jp-api.spooncast.net/lives/%v/", m.liveId)
	webText, err = conn.GetWebPage(ctx, url)
	if err != nil {
		return err
	}
	if !m.parseLiveInfoPage(webText) {
		return inter.ErrNolive
	}

	err = m.LoadJsVersion(ctx, conn)
	if err != nil {
		log.Println("find js appversion fail, use default value,", err)
		m.jsAppVersion = "8.0.1"
	}

	log.Println("m3u8 url:", m.M3u8Url)
	log.Println("js appversion:", m.jsAppVersion)

	return nil
}

func (m *Spoon) LoadJsVersion(ctx context.Context, conn inter.INet) error {
	url := fmt.Sprintf(`https://www.spooncast.net/jp/live/@%v`, m.Id)
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
	m.jsAppVersion = match[1]
	return nil
}

type jmap = map[string]any

func (m *Spoon) parseMetaPage(jsonText string) bool {
	var jsonmap jmap
	decoder := json.NewDecoder(strings.NewReader(jsonText))
	decoder.UseNumber()
	decoder.Decode(&jsonmap)
	//json.Unmarshal([]byte(jsonText), &jsonmap)
	for _, result := range jsonmap["results"].([]any) {
		if liveId, exist := result.(jmap)["current_live_id"]; exist {
			if liveId != nil { // nil if offline
				m.liveId = fmt.Sprint(liveId)
				m.Chat = fmt.Sprintf(`wss://jp-heimdallr.spooncast.net/%v`, liveId)
			}
		}
	}
	return m.liveId != ""
}

func (m *Spoon) parseLiveInfoPage(jsonText string) bool {
	var jsonmap jmap
	json.Unmarshal([]byte(jsonText), &jsonmap)
	for _, result := range jsonmap["results"].([]any) {
		if url_hls, exist := result.(jmap)["url_hls"]; exist {
			m.M3u8Url = url_hls.(string)
		}
		if img_url, exist := result.(jmap)["img_url"]; exist {
			m.imgUrl = img_url.(string)
		}
		if title, exist := result.(jmap)["title"]; exist {
			m.title = title.(string)
		}
		if author, exist := result.(jmap)["author"]; exist {
			if nickname, exist := author.(jmap)["nickname"]; exist {
				m.Name = nickname.(string)
			}
		}
	}
	return m.M3u8Url != ""
}

func (m *Spoon) Download(ctx context.Context, netconn inter.INet, fio inter.IFs, filename string) error {
	ctx2, cancel := context.WithCancel(ctx)
	chat := &Chat{
		Fs:           fio,
		liveId:       m.liveId,
		jsAppVersion: m.jsAppVersion,
	}
	var cs chan int
	if len(m.Chat) > 0 {
		cs = make(chan int, 1)
		go func() {
			chat.Connect(ctx2, m.Chat, filename)
			cs <- 1
		}()
	}

	coverCh := make(chan string, 1)
	if m.imgUrl == "" {
		coverCh <- ""
	} else {
		coverFileName, err := inter.DownloadThumbnail(ctx, netconn, fio, filename, m.imgUrl)
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
	m.vd = &m3u8.Downloader{
		Chat:     chat,
		GuessTs:  guessTs,
		IgnoreTs: ignorePattern,
	}
	m.vd.DownloadMerge(ctx, m.M3u8Url, netconn, fio, filename)
	cancel()
	if cs != nil {
		<-cs
	}
	if m.vd.GetFragCount() == 0 {
		return inter.ErrNolive
	}

	coverFile := <-coverCh
	if coverFile != "" {
		inter.FfmpegAttachThumbnail(filename+".mp4", coverFile, 1)
	}
	if m.title != "" {
		meta := inter.FfmpegMeta{
			Title:  m.title,
			Artist: m.Name,
			Album:  fmt.Sprintf("%v-%v", m.Name, m.title),
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

package mixch

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

type Mixch struct {
	ImageUrl string
	Name     string
	Id       string
	M3u8Url  string
	Chat     string
	vd       *m3u8.Downloader
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

func (m *Mixch) WaitStreamStart(ctx context.Context, conn inter.INet) error {
	err := m.LoadUserPage(ctx, conn)
	if errors.Is(err, inter.ErrNolive) {
		log.Println("wait stream start......")
		err = m.waitLiveLoop(ctx, 15*time.Second, conn)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (m *Mixch) waitLiveLoop(ctx context.Context, interval time.Duration, conn inter.INet) error {
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

func (m *Mixch) LoadUserPage(ctx context.Context, conn inter.INet) error {
	url := fmt.Sprintf("https://mixch.tv/u/%v/live", m.Id)
	webText, err := conn.GetWebPage(ctx, url)
	if err != nil {
		return err
	}

	if !m.parseLivePage(webText) || len(m.M3u8Url) == 0 {
		return inter.ErrNolive
	}

	log.Println("m3u8 url:", m.M3u8Url)

	return nil
}

func (m *Mixch) Download(ctx context.Context, netconn inter.INet, fio inter.IFs, filename string) error {
	ctx2, cancel := context.WithCancel(ctx)
	chat := &Chat{Fs: fio}
	var cs chan int
	if len(m.Chat) > 0 {
		cs = make(chan int, 1)
		go func() {
			chat.Connect(ctx2, m.Chat, filename)
			cs <- 1
		}()
	}

	m.vd = &m3u8.Downloader{
		Chat:    chat,
		GuessTs: guessTs,
	}
	m.vd.DownloadMerge(ctx, m.M3u8Url, netconn, fio, filename)
	cancel()
	if cs != nil {
		<-cs
	}
	if m.vd.GetFragCount() == 0 {
		return inter.ErrNolive
	} else {
		generateHtml(filename + ".mp4")
		return nil
	}
}

func generateHtml(mp4 string) {
	option := cplayer.NewOption()
	cplayer.ProcessVideo(option, mp4)
}

func (m *Mixch) parseLivePage(htmContent string) bool {
	prefix := "window.__INITIAL_JS_STATE__ = "
	for _, line := range strings.Split(htmContent, "\n") {
		if strings.HasPrefix(line, prefix) {
			text := strings.TrimPrefix(line, prefix)
			text = strings.TrimSuffix(text, ";")
			return m.parseLivePageJson(text)
		}
	}
	return false
}

type jmap = map[string]any

func (m *Mixch) parseLivePageJson(jsonText string) bool {
	var jsonmap jmap
	json.Unmarshal([]byte(jsonText), &jsonmap)
	liveInfo, exist := jsonmap["liveInfo"].(jmap)
	if exist {
		hls, exist := liveInfo["hls"]
		if !exist {
			return false
		}
		m.M3u8Url = hls.(string)
		chat, exist := liveInfo["chat"]
		if exist {
			m.Chat = chat.(string)
		}
	}
	broInfo, exist := jsonmap["broadcasterInfo"].(jmap)
	if exist {
		name, exist := broInfo["name"].(string)
		if exist {
			m.Name = name
		}
		imageUrl, exist := broInfo["profile_image_url"].(string)
		if exist {
			m.ImageUrl = imageUrl
		}
	}

	return true
}

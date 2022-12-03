package twitcasting

import (
	"context"
	"errors"
	"fmt"
	"github.com/ting1322/chat-player/pkg/cplayer"
	"inter"
	"log"
	"strings"
	"time"
)

type jmap = map[string]any

type Live struct {
	//ImageUrl string
	//Name     string
	IsLive  bool
	Id      string
	MovieId string
	//M3u8Url  string
	VideoUrl string
	Chat     string
	MainPage string
}

func New(text string) *Live {
	id := strings.TrimPrefix(text, "https://twitcasting.tv/")
	if strings.Index(id, "/") > 0 {
		id = id[0:strings.Index(id, "/")]
	}
	live := &Live{MainPage: text, Id: id}
	return live
}

func (m *Live) WaitStreamStart(ctx context.Context, conn inter.INet) error {
	err := m.LoadUserPage(ctx, conn)
	if errors.Is(err, inter.ErrNolive) {
		err = m.waitLiveLoop(ctx, conn)
		if err != nil {
			log.Fatal(err)
		}
	} else if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (m *Live) waitLiveLoop(ctx context.Context, conn inter.INet) error {
	timer := time.NewTimer(30 * time.Second)
	for {
		log.Println("no live, retry after 30s")
		<-timer.C
		err := m.LoadUserPage(ctx, conn)
		if err == nil {
			log.Println("live start.")
			return nil
		}
		if !errors.Is(err, inter.ErrNolive) {
			return fmt.Errorf("wait live start: %w", err)
		}
		timer.Reset(30 * time.Second)
	}
}

func (m *Live) LoadUserPage(ctx context.Context, conn inter.INet) error {
	videoInfoUrl := fmt.Sprintf("https://twitcasting.tv/streamserver.php?target=%v&mode=client", m.Id)
	webText, err := conn.GetWebPage(ctx, videoInfoUrl)
	if err != nil {
		return fmt.Errorf("get video info: %w", err)
	}

	err = parseStreamInfo(m, webText)
	if err != nil {
		log.Printf("Response: %v\n", webText)
		return fmt.Errorf("parse video info: %w", err)
	}

	pdata := make(map[string]string)
	pdata["movie_id"] = m.MovieId
	webText, err = conn.Post(ctx, "https://twitcasting.tv/eventpubsuburl.php", pdata)
	if err != nil {
		return fmt.Errorf("get chat info: %w", err)
	}
	err = parseChatInfo(m, webText)
	if err != nil {
		log.Printf("Response: %v\n", webText)
		return fmt.Errorf("parse chat info: %w", err)
	}
	if !m.IsLive || len(m.VideoUrl) == 0 || len(m.Chat) == 0 {
		return inter.ErrNolive
	}
	return nil
}

func (m *Live) Download(ctx context.Context, netconn inter.INet, fio inter.IFs, filename string) {
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

	vd := &VDown{
		fs:   fio,
		conn: netconn,
	}
	vd.DownloadMerge(ctx, m.VideoUrl, filename)
	cancel()
	if cs != nil {
		<-cs
	}
	generateHtml(filename + ".mp4")
}

func generateHtml(mp4 string) {
	option := cplayer.NewOption()
	cplayer.ProcessVideo(option, mp4)
}

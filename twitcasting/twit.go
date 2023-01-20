package twitcasting

import (
	"context"
	"errors"
	"fmt"
	"inter"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ting1322/chat-player/pkg/cplayer"
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
	vd       *VDown
	pass     string // user password text
	wpass    string // cookie password sent from server
}

func New(text, pass string) *Live {
	id := strings.TrimPrefix(text, "https://twitcasting.tv/")
	if strings.Index(id, "/") > 0 {
		id = id[0:strings.Index(id, "/")]
	}
	live := &Live{MainPage: text, Id: id, pass: pass}
	return live
}

func (m *Live) WaitStreamStart(ctx context.Context, conn inter.INet) error {
	err := m.LoadUserPage(ctx, conn)
	if errors.Is(err, inter.ErrNolive) {
		log.Println("wait stream start......")
		err = m.waitLiveLoop(ctx, 10*time.Second, conn)
		if err != nil {
			log.Fatal(err)
		}
	} else if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (m *Live) waitLiveLoop(ctx context.Context, interval time.Duration, conn inter.INet) error {
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

func (m *Live) LoadUserPage(ctx context.Context, conn inter.INet) error {
	if m.pass != "" {
		userInfoUrl := fmt.Sprintf("https://twitcasting.tv/%v", m.Id)
		webText, err := conn.GetWebPage(ctx, userInfoUrl)
		if err != nil {
			return fmt.Errorf("get user page: %w", err)
		}

		if strings.Contains(webText, `<input type="text" name="password" value="">`) {
			//if m.pass == "" {
			//	return errors.New("password word is required")
			//}
			postData := make(map[string]string)
			postData["password"] = url.QueryEscape(m.pass)
			re, err := regexp.Compile(`<input type="hidden" name="cs_session_id" value="(\w+)">`)
			mgroup := re.FindStringSubmatch(webText)
			if mgroup != nil {
				postData["cs_session_id"] = mgroup[1]
			}
			webText, err = conn.Post(ctx, userInfoUrl, postData)
			if err != nil {
				return fmt.Errorf("submit password: %w", err)
			}
			m.wpass, err = conn.GetCookie("wpass", "https://twitcasting.tv", "/"+m.Id)
			if err != nil {
				return fmt.Errorf("get password: %w", err)
			}
		}
	}

	videoInfoUrl := fmt.Sprintf("https://twitcasting.tv/streamserver.php?target=%v&mode=client", m.Id)
	webText, err := conn.GetWebPage(ctx, videoInfoUrl)
	if err != nil {
		return fmt.Errorf("get video info: %w", err)
	}

	err = parseStreamInfo(m, webText)
	if err != nil {
		return fmt.Errorf("parse video info: %w\nresponse: %v", err, webText)
	}

	pdata := make(map[string]string)
	pdata["movie_id"] = m.MovieId
	if m.wpass != "" {
		pdata["password"] = m.wpass
		log.Println("use password:", m.wpass)
	}
	webText, err = conn.PostForm(ctx, "https://twitcasting.tv/eventpubsuburl.php", pdata)
	if err != nil {
		return fmt.Errorf("get chat info: %w", err)
	}
	err = parseChatInfo(m, webText)
	if err != nil {
		return fmt.Errorf("parse chat info: %w\nresponse: %v", err, webText)
	}
	if !m.IsLive || len(m.VideoUrl) == 0 || len(m.Chat) == 0 {
		return inter.ErrNolive
	}
	return nil
}

func (m *Live) Download(ctx context.Context, netconn inter.INet, fio inter.IFs, filename string) error {
	ctx2, cancel := context.WithCancel(ctx)
	chat := &Chat{Fs: fio}
	var cs chan int
	if len(m.Chat) > 0 {
		cs = make(chan int, 1)
		go func() {
			chat.Connect(ctx2, netconn, m.Chat, filename)
			cs <- 1
		}()
	}

	m.vd = &VDown{
		fs:   fio,
		conn: netconn,
		chat: chat,
	}
	m.vd.DownloadMerge(ctx, netconn, m.VideoUrl, filename)
	cancel()
	if cs != nil {
		<-cs
	}
	generateHtml(filename + ".mp4")
	return nil
}

func generateHtml(mp4 string) {
	option := cplayer.NewOption()
	cplayer.ProcessVideo(option, mp4)
}

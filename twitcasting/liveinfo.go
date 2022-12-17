package twitcasting

import (
	"encoding/json"
	"errors"
	"inter"
	"log"
	"regexp"
)

var ErrFormat error = errors.New("format error")

func parseStreamInfo(info *Live, text string) error {
	re, _ := regexp.Compile(`"movie":\{"id":(\d+),`)
	match := re.FindStringSubmatch(text)
	if len(match) >= 2 {
		info.MovieId = match[1]
	}
	var jsonmap jmap
	json.Unmarshal([]byte(text), &jsonmap)
	if movie, exist := jsonmap["movie"].(jmap); exist {
		if live, exist := movie["live"].(bool); exist {
			info.IsLive = live
		}
	}
	if llfmp4, exist := jsonmap["llfmp4"].(jmap); exist {
		if streams, exist := llfmp4["streams"].(jmap); exist {
			if main, exist := streams["main"].(string); exist {
				info.VideoUrl = main
			}
		}
	}
	if !info.IsLive {
		return inter.ErrNolive
	}
	if info.wpass != "" && info.VideoUrl != "" {
		info.VideoUrl += "?word=" + info.wpass
	}
	return nil
}

type _ChatJsonInfo struct {
	Url string `json:"url"`
}

func parseChatInfo(info *Live, text string) error {
	m := _ChatJsonInfo{}
	err := json.Unmarshal([]byte(text), &m)
	if err != nil {
		log.Println(text)
		return err
	}
	info.Chat = m.Url
	return nil
}

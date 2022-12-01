package mixch

import (
	"context"
	"encoding/json"
	"fmt"
	"inter"
	"io"
	"log"
	"time"
	"nhooyr.io/websocket"
)

type Chat struct {
	Fs   inter.IFs
	Msgs []Record
}

type Record struct {
	Time   time.Time
	Author string
	Msg    string
}

func (chat *Chat) Connect(ctx context.Context, wssUrl string, liveName string) {
	writer, err := chat.Fs.Create(liveName + ".live_chat.json")
	if err != nil {
		log.Println(err)
		return
	}
	defer writer.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			chat.connectTry1(ctx, wssUrl, writer)
		}
	}
}

func (chat *Chat) connectTry1(ctx context.Context, wssUrl string, writer io.Writer) {
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	log.Println("WSS:", wssUrl)
	c, _, err := websocket.Dial(ctx2, wssUrl, nil)
	cancel()
	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		c.Close(websocket.StatusNormalClosure, "")
		log.Println("WSS: close")
	}()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			log.Println("try1 done")
			return
		default:
			ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, data, err := c.Read(ctx2)
			cancel()
			if err != nil {
				log.Println("mixch/chat connectTry1", err)
				return
			}

			var jsonmap jmap
			json.Unmarshal(data, &jsonmap)
			if kind, exist := jsonmap["kind"]; exist {
				if kind.(float64) == 0 {
					msgTime := time.Since(startTime).Milliseconds()
					name := jsonmap["name"].(string)
					body := jsonmap["body"].(string)
					ytc := ConvertToYtChat(msgTime, name, body)
					writer.Write(ytc)
					writer.Write([]byte("\n"))
				}
			}
		}
	}

}

func ConvertToYtChat(msgTime int64, name, body string) []byte {

	ytmap := jmap{}
	replayChatItemAction := jmap{}
	ytmap["replayChatItemAction"] = replayChatItemAction
	r2 := make([]jmap, 1)
	r2[0] = jmap{}
	replayChatItemAction["actions"] = r2
	r3 := jmap{}
	r2[0]["addChatItemAction"] = r3
	r4 := jmap{}
	r3["item"] = r4
	liveChatTextMessageRenderer := jmap{}
	r4["liveChatTextMessageRenderer"] = liveChatTextMessageRenderer
	r6 := jmap{}
	liveChatTextMessageRenderer["message"] = r6
	r7 := make([]jmap, 1)
	r6["runs"] = r7
	r8 := jmap{}
	r7[0] = r8
	r8["text"] = body

	aurhorName := jmap{}
	liveChatTextMessageRenderer["authorName"] = aurhorName
	aurhorName["simpleText"] = name
	replayChatItemAction["videoOffsetTimeMsec"] = fmt.Sprintf("%v", msgTime)
	data, _ := json.Marshal(ytmap)
	return data
}

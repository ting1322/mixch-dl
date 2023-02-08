package spoon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"inter"
	"io"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type Chat struct {
	liveId    string
	Fs        inter.IFs
	startTime time.Time
	count     int
	mu        sync.Mutex
}

type Record struct {
	Time   time.Time
	Author string
	Msg    string
}

func (chat *Chat) SetTime(t time.Duration) {
	chat.mu.Lock()
	chat.startTime = time.Now().Add(-t)
	chat.mu.Unlock()
}

func (chat *Chat) getStartTime() time.Time {
	chat.mu.Lock()
	defer chat.mu.Unlock()
	return chat.startTime
}

func (chat *Chat) Count() int {
	chat.mu.Lock()
	defer chat.mu.Unlock()
	return chat.count
}

func (chat *Chat) incCount() {
	chat.mu.Lock()
	chat.count++
	chat.mu.Unlock()
}

func (chat *Chat) Connect(ctx context.Context, wssUrl string, liveName string) {
	writer, err := chat.Fs.Create(liveName + ".live_chat.json")
	if err != nil {
		log.Println(err)
		return
	}
	defer writer.Close()

	chat.SetTime(0)

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

	initMsg := fmt.Sprintf(`{"live_id": "%v", "appversion": "8.0.1", "retry": 0, "reconnect": false, "event": "live_join", "type": "live_req", "useragent": "Web"}`, chat.liveId)

	keepMsg := fmt.Sprintf(`{"live_id": "%v", "appversion": "8.0.1", "event": "live_health", "type": "live_rpt", "useragent": "Web"}`, chat.liveId)

	ctx2, cancel = context.WithTimeout(ctx, 15*time.Second)
	c.Write(ctx2, websocket.MessageText, []byte(initMsg))
	cancel()


	for {
		select {
		case <-ctx.Done():
			log.Println("try1 done")
			return

		default:
			ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, data, err := c.Read(ctx2)
			cancel()
			if ctx.Err() == context.Canceled {
				return
			} else if err != nil {
				log.Println("mixch/chat connectTry1", err)
				return
			}
			//log.Println("(wss chat)", "receive", string(data))

			var jsonmap jmap
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.UseNumber()
			decoder.Decode(&jsonmap)
			if event, exist := jsonmap["event"]; exist {
				evtStr := event.(string)
				if evtStr == "live_health" {
					log.Println("(wss chat)", "keep alive")
					ctx2, cancel = context.WithTimeout(ctx, 15*time.Second)
					c.Write(ctx2, websocket.MessageText, []byte(keepMsg))
					cancel()
				} else if evtStr == "live_message" {
					userName, body, success := decodeLiveMessage(jsonmap)
					log.Println("(wss chat)", userName, body)
					if success {
						msgTime := time.Since(chat.getStartTime()).Milliseconds()
						ytc := ConvertToYtChat(msgTime, userName, body)
						writer.Write(ytc)
						writer.Write([]byte("\n"))
						chat.incCount()
					}
				}
			}
		}
	}

}


func decodeLiveMessage(jsonmap jmap) (userName, body string, success bool) {
	data, exist := jsonmap["data"]
	if !exist {
		return "", "", false
	}
	user, exist := data.(jmap)["user"]
	if !exist {
		return "", "", false
	}
	nickName, exist := user.(jmap)["nickname"]
	if !exist {
		return "", "", false
	}
	userName = nickName.(string)
	update_component, exist := jsonmap["update_component"]
	if !exist {
		return "", "", false
	}
	message, exist := update_component.(jmap)["message"]
	if !exist {
		return "", "", false
	}
	value, exist := message.(jmap)["value"]
	if !exist {
		return "", "", false
	}
	body = value.(string)
	return userName, body, true
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

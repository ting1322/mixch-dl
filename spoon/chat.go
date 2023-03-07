package spoon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"inter"
	"io"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type Chat struct {
	liveId       string
	jsAppVersion string
	Fs           inter.IFs
	startTime    time.Time
	count        int
	mu           sync.Mutex
}

type Record struct {
	Time   time.Time
	Author string
	Msg    string
}

func (this *Chat) SetTime(t time.Duration) {
	this.mu.Lock()
	this.startTime = time.Now().Add(-t)
	this.mu.Unlock()
}

func (this *Chat) getStartTime() time.Time {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.startTime
}

func (this *Chat) Count() int {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.count
}

func (this *Chat) incCount() {
	this.mu.Lock()
	this.count++
	this.mu.Unlock()
}

func (this *Chat) Connect(ctx context.Context, wssUrl string, liveName string) {
	writer, err := this.Fs.Create(liveName + ".live_chat.json")
	if err != nil {
		inter.LogMsg(false, fmt.Sprintf("WSS (chat): in Connect: %v", err))
		return
	}
	defer writer.Close()

	this.SetTime(0)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			this.connectTry1(ctx, wssUrl, writer)
		}
	}
}

func (this *Chat) connectTry1(ctx context.Context, wssUrl string, writer io.Writer) {
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	inter.LogMsg(false, fmt.Sprintf("WSS (chat): %v", wssUrl))
	c, _, err := websocket.Dial(ctx2, wssUrl, nil)
	cancel()
	if err != nil {
		inter.LogMsg(false, fmt.Sprintf("WSS (chat): in connectTry1: %v", err))
		return
	}

	defer func() {
		c.Close(websocket.StatusNormalClosure, "")
		inter.LogMsg(false, "WSS (chat): close")
	}()

	initMsg := fmt.Sprintf(`{"live_id": "%v", "appversion": "%v", "retry": 0, "reconnect": false, "event": "live_join", "type": "live_req", "useragent": "Web"}`, this.liveId, this.jsAppVersion)

	keepMsg := fmt.Sprintf(`{"live_id": "%v", "appversion": "%v", "event": "live_health", "type": "live_rpt", "useragent": "Web"}`, this.liveId, this.jsAppVersion)

	ctx2, cancel = context.WithTimeout(ctx, 15*time.Second)
	c.Write(ctx2, websocket.MessageText, []byte(initMsg))
	cancel()

	for {
		select {
		case <-ctx.Done():
			inter.LogMsg(false, "WSS (chat): in connectTry1: done")
			return

		default:
			ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, data, err := c.Read(ctx2)
			cancel()
			if ctx.Err() == context.Canceled {
				return
			} else if err != nil {
				inter.LogMsg(false, fmt.Sprintf("WSS (chat): in connectTry1: %v", err))
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
					inter.LogMsg(false, "WSS (chat): keep alive")
					ctx2, cancel = context.WithTimeout(ctx, 15*time.Second)
					c.Write(ctx2, websocket.MessageText, []byte(keepMsg))
					cancel()
				} else if evtStr == "live_message" {
					userName, body, success := decodeLiveMessage(jsonmap)
					inter.LogMsg(false, fmt.Sprintf("WSS (chat): msg: %v: %v", userName, body))
					if success {
						msgTime := time.Since(this.getStartTime()).Milliseconds()
						ytc := ConvertToYtChat(msgTime, userName, body)
						writer.Write(ytc)
						writer.Write([]byte("\n"))
						this.incCount()
					}
				} else if evtStr == "use_item" {
					inter.LogMsg(true, "WSS (chat): use_item")
					// skip
				} else if evtStr == "lazy_update" ||
					evtStr == "live_like" ||
					evtStr == "live_rank" ||
					evtStr == "live_join" ||
					evtStr == "live_update" {
					// skip
				} else {
					inter.LogMsg(true, fmt.Sprintf("WSS (chat): unknown msg: %v", string(data)))
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

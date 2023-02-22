package twitcasting

import (
	"context"
	"encoding/json"
	"inter"
	"io"
	"log"
	"mixch"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type Chat struct {
	Fs    inter.IFs
	mu    sync.Mutex
	count int64
}

func (this *Chat) GetCount() int64 {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.count
}

func (this *Chat) incCount() {
	this.mu.Lock()
	this.count++
	this.mu.Unlock()
}

func (this *Chat) Connect(ctx context.Context, netconn inter.INet, wssurl, filename string) {
	writer, err := this.Fs.Create(filename + ".live_chat.json")
	if err != nil {
		log.Println(err)
		return
	}
	defer writer.Close()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			this.connectTry1(ctx, netconn, wssurl, writer, startTime)
		}
	}

}

func (this *Chat) connectTry1(ctx context.Context, netconn inter.INet, wssUrl string, writer io.Writer, startTime time.Time) {
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	log.Println("WSS (chat):", wssUrl)
	dopt := &websocket.DialOptions{
		HTTPClient: netconn.GetHttpClient(),
	}
	c, _, err := websocket.Dial(ctx2, wssUrl, dopt)
	cancel()
	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		c.Close(websocket.StatusNormalClosure, "")
		log.Println("WSS (chat): close")
	}()

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
				log.Println("connectTry1", err)
				return
			}

			msgTime := time.Since(startTime).Milliseconds()
			this.parseChatData(data, writer, msgTime)
		}
	}
}

func (this *Chat) parseChatData(data []byte, writer io.Writer, msgTime int64) {
	var root []jmap
	json.Unmarshal(data, &root)
	for _, node := range root {
		if t, exist := node["type"]; exist {
			if t.(string) == "comment" {
				this.parseComment(node, writer, msgTime)
			}
		}
	}
}

func (this *Chat) parseComment(node jmap, writer io.Writer, msgTime int64) {
	message, exist := node["message"]
	if !exist {
		log.Println("not found message")
		return
	}
	body := message.(string)
	author, exist := node["author"]
	if !exist {
		log.Println("not found author")
		return
	}
	this.incCount()
	name := author.(jmap)["name"].(string)

	ytc := mixch.ConvertToYtChat(msgTime, name, body)
	writer.Write(ytc)
	writer.Write([]byte("\n"))
}

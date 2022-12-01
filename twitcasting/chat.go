package twitcasting

import (
	"context"
	"encoding/json"
	"inter"
	"io"
	"log"
	"mixch"
	"time"

	"nhooyr.io/websocket"
)

type Chat struct {
	Fs inter.IFs
}

func (c *Chat) Connect(ctx context.Context, wssurl string, filename string) {
	writer, err := c.Fs.Create(filename + ".live_chat.json")
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
			c.connectTry1(ctx, wssurl, writer)
		}
	}

}

func (chat *Chat) connectTry1(ctx context.Context, wssUrl string, writer io.Writer) {
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	log.Println("WSS (chat):", wssUrl)
	c, _, err := websocket.Dial(ctx2, wssUrl, nil)
	cancel()
	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		c.Close(websocket.StatusNormalClosure, "")
		log.Println("WSS (chat): close")
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
				log.Println("connectTry1", err)
				return
			}

			msgTime := time.Since(startTime).Milliseconds()
			chat.parseChatData(data, writer, msgTime)
		}
	}
}

func (chat *Chat) parseChatData(data []byte, writer io.Writer, msgTime int64) {
	var root []jmap
	json.Unmarshal(data, &root)
	for _, node := range root {
		if t, exist := node["type"]; exist {
			if t.(string) == "comment" {
				chat.parseComment(node, writer, msgTime)
			}
		}
	}
}

func (chat *Chat) parseComment(node jmap, writer io.Writer, msgTime int64) {
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
	name := author.(jmap)["name"].(string)

	ytc := mixch.ConvertToYtChat(msgTime, name, body)
	writer.Write(ytc)
	writer.Write([]byte("\n"))
}

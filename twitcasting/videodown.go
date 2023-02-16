package twitcasting

import (
	"context"
	"fmt"
	"inter"
	"io"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type VDown struct {
	fs        inter.IFs
	conn      inter.INet
	fragCount int64
	mu        sync.Mutex
	chat      *Chat
}

type Status int

const (
	SomeSuccess Status = iota
	AllFail
)

func (v *VDown) DownloadMerge(ctx context.Context, netconn inter.INet, wssurl string, filename string) {
	tspartFilename := filename + ".ts.part"
	v.downloadLoop(ctx, netconn, wssurl, tspartFilename)
	if v.fs.Exist(tspartFilename) {
		inter.FfmpegMerge(tspartFilename, filename+".mp4", true)
		inter.FfmpegFastStartMp4(filename + ".mp4")
	}
}

func (v *VDown) GetFragCount() int64 {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.fragCount
}

func (v *VDown) incFrag() {
	v.mu.Lock()
	v.fragCount++
	v.mu.Unlock()
}

func (v *VDown) downloadLoop(ctx context.Context, netconn inter.INet, wssurl, filename string) {
	writer, err := v.fs.Create(filename)
	if err != nil {
		log.Println("downloadLoop", err)
		return
	}
	defer writer.Close()

	retry := 5
	for {
		select {
		case <-ctx.Done():
			return
		default:
			status, err := v.try1(ctx, netconn, wssurl, writer) // return until error
			if ctx.Err() == context.Canceled {
				// nothing
			} else if err != nil {
				log.Println()
				log.Printf("WSS(video) error, retry=%v, %v\n", retry, err)
				if status == AllFail {
					retry--
				} else if status == SomeSuccess {
					retry = 5
				}
				if retry <= 0 {
					return
				}
			}
		}
	}
}
func (v *VDown) try1(ctx context.Context, netconn inter.INet, wssurl string, writer io.Writer) (Status, error) {
	var status Status = AllFail
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	log.Println("WSS(video):", wssurl)
	dopt := &websocket.DialOptions{
		HTTPClient: netconn.GetHttpClient(),
	}

	c, _, err := websocket.Dial(ctx2, wssurl, dopt)
	cancel()
	if err != nil {
		log.Println(err)
		return status, fmt.Errorf("open websocket: %w", err)
	}
	c.SetReadLimit(1024 * 1024 * 4)

	defer func() {
		c.Close(websocket.StatusNormalClosure, "")
		log.Println("WSS(video): close")
	}()

	statusTimer := time.NewTimer(2 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return status, nil
		case <-statusTimer.C:
			inter.ClearLine()
			fmt.Printf("downloaded video fragment: %d, chat: %d\r", v.GetFragCount(), v.chat.GetCount())
			statusTimer.Reset(2 * time.Second)
			break
		default:
			ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, data, err := c.Read(ctx2)
			cancel()
			if err != nil {
				return status, fmt.Errorf("read websocket: %w", err)
			}
			v.incFrag()
			status = SomeSuccess
			writer.Write(data)
		}
	}
}

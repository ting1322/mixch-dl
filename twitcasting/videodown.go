package twitcasting

import (
	"context"
	"fmt"
	"mixch-dl/inter"
	"io"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type VDown struct {
	fs        inter.IFs
	conn      inter.INet
	fragCount int
	mu        sync.Mutex
	chat      *Chat
}

type Status int

const (
	SomeSuccess Status = iota
	AllFail
)

func (this *VDown) DownloadMerge(ctx context.Context, netconn inter.INet, wssurl string, filename string) {
	tspartFilename := filename + ".ts.part"
	this.downloadLoop(ctx, netconn, wssurl, tspartFilename)
	if this.fs.Exist(tspartFilename) {
		inter.FfmpegMerge(tspartFilename, filename+".mp4", true)
		inter.FfmpegFastStartMp4(filename + ".mp4")
	}
}

func (this *VDown) GetFragCount() int {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.fragCount
}

func (this *VDown) incFrag() {
	this.mu.Lock()
	this.fragCount++
	this.mu.Unlock()
}

func (this *VDown) downloadLoop(ctx context.Context, netconn inter.INet, wssurl, filename string) {
	writer, err := this.fs.Create(filename)
	if err != nil {
		inter.LogMsg(false, fmt.Sprint("create file error: ", err))
		return
	}
	defer writer.Close()

	retry := 5
	for {
		select {
		case <-ctx.Done():
			return
		default:
			status, err := this.try1(ctx, netconn, wssurl, writer) // return until error
			if ctx.Err() == context.Canceled {
				// nothing
			} else if err != nil {
				inter.LogMsg(false, fmt.Sprintf("WSS(video): error, retry=%v, %v\n", retry, err))
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
func (this *VDown) try1(ctx context.Context, netconn inter.INet, wssurl string, writer io.Writer) (Status, error) {
	var status Status = AllFail
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	inter.LogMsg(false, fmt.Sprintf("WSS(video): %v", wssurl))
	dopt := &websocket.DialOptions{
		HTTPClient: netconn.GetHttpClient(),
	}

	c, _, err := websocket.Dial(ctx2, wssurl, dopt)
	cancel()
	if err != nil {
		inter.LogMsg(false, fmt.Sprintf("WSS(video): %v", err))
		return status, fmt.Errorf("open websocket: %w", err)
	}
	c.SetReadLimit(1024 * 1024 * 4)

	defer func() {
		c.Close(websocket.StatusNormalClosure, "")
		inter.LogMsg(false, fmt.Sprintf("WSS(video): close"))
	}()

	statusTimer := time.NewTimer(2 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return status, nil
		case <-statusTimer.C:
			inter.ClearLine()
			inter.LogProgress(this.GetFragCount(), this.chat.GetCount(), 0)
			statusTimer.Reset(2 * time.Second)
			break
		default:
			ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, data, err := c.Read(ctx2)
			cancel()
			if err != nil {
				return status, fmt.Errorf("read websocket: %w", err)
			}
			this.incFrag()
			status = SomeSuccess
			writer.Write(data)
		}
	}
}

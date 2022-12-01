package twitcasting

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"
	"inter"

	"nhooyr.io/websocket"
)

type VDown struct {
	fs   inter.IFs
	conn inter.INet
}

func (v *VDown) DownloadMerge(ctx context.Context, wssurl string, filename string) {
	tspartFilename := filename + ".ts.part"
	v.downloadLoop(ctx, wssurl, tspartFilename)
	if v.fs.Exist(tspartFilename) {
		inter.FfmpegMerge(tspartFilename, filename+".mp4", true)
	}
}
func (v *VDown) downloadLoop(ctx context.Context, wssurl, filename string) {
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
			err := v.try1(ctx, wssurl, writer)
			if err != nil {
				retry--
				log.Printf("WSS(video) error, retry=%v, %v\n", retry, err)
				if retry <= 0 {
					return
				}
			} else {
				retry = 5
			}
		}
	}
}
func (v *VDown) try1(ctx context.Context, wssurl string, writer io.Writer) error {
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	log.Println("WSS(video):", wssurl)
	c, _, err := websocket.Dial(ctx2, wssurl, nil)
	c.SetReadLimit(1024 * 1024 * 4)
	cancel()
	if err != nil {
		log.Println(err)
		return fmt.Errorf("open websocket: %w", err)
	}

	defer func() {
		c.Close(websocket.StatusNormalClosure, "")
		log.Println("WSS(video): close")
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, data, err := c.Read(ctx2)
			cancel()
			if err != nil {
				return fmt.Errorf("read websocket: %w", err)
			}

			writer.Write(data)
		}
	}
}

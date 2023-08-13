package m3u8

import (
	"context"
	"github.com/ting1322/mixch-dl/inter"
)

type DownloadWorker struct {
	url      string
	complete chan DownResult
}

type DownResult struct {
	err  error
	data []byte
}

func NewWorker(url string) *DownloadWorker {
	w := &DownloadWorker{url: url}
	w.complete = make(chan DownResult, 1)
	return w
}

func (this *DownloadWorker) run(ctx context.Context, conn inter.INet) {
	data, err := conn.GetFile(ctx, this.url)
	result := DownResult{err, data}
	this.complete <- result
}

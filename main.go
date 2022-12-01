package main

import (
	"context"
	"fmt"
	"inter"
	"log"
	"m3u8"
	"mixch"
	"os"
	"os/signal"
	"time"
	"twitcasting"
)

var (
	downloader *m3u8.Downloader = &m3u8.Downloader{}
	fio        inter.IFs        = &inter.Fs{}
	netconn    inter.INet       = &inter.Net{}
)

func waitLiveStart(ctx context.Context, live *mixch.Mixch) error {
	timer := time.NewTimer(30 * time.Second)
	for {
		log.Println("no live, retry after 30s")
		<-timer.C
		err := live.LoadUserPage(ctx, netconn)
		if err == nil {
			log.Println("live start.")
			return nil
		}
		if err != inter.ErrNolive {
			return err
		}
		timer.Reset(30 * time.Second)
	}
}

func parseTime(text string) (time.Time, error) {
	now := time.Now().Local()
	t, err := time.ParseInLocation("15:04", os.Args[2], time.Now().Location())
	if err == nil {
		y, m, d := now.Date()
		t = t.AddDate(y, int(m) - 1, d - 1)
		if t.Before(now) {
			t = t.AddDate(0, 0, 1)
		}
		return t, nil
	}
	return time.Now(), err
}

func main() {
	var url string
	if len(os.Args) > 1 {
		url = os.Args[1]
	} else {
		log.Printf(`
need a url as argument, for example:
  mixch-dl https://mixch.tv/u/17209506

  mixch-dl https://mixch.tv/u/17209506/live
    (url with /live is allowed)

  mixch-dl https://mixch.tv/u/17209506  18:57
    (wait until 18:57)
`)
	}
	if len(os.Args) > 2 {
		t, err := parseTime(os.Args[2])
		if err != nil {
			log.Fatal("time format error", os.Args[2], err)
		}
		fmt.Printf("wait until %v, (%ds)", t, int(t.Sub(time.Now().Local()).Seconds()))
		timer := time.NewTimer(5 * time.Second)
		for t.After(time.Now().Local()) {
			timer.Reset(5 * time.Second)
			<-timer.C
			fmt.Printf("\rwait until %v, (%ds)", t, int(t.Sub(time.Now().Local()).Seconds()))
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var filename string
	var live inter.Live
	if mixch.Support(url) {
		filename = fmt.Sprintf("mixch-%v", time.Now().Local().Format("2006-01-02-15-04"))
		mlive, err := mixch.New(url)
		live = mlive
		if err != nil {
			log.Fatal(err)
		}
		err = mlive.LoadUserPage(ctx, netconn)
		if err == inter.ErrNolive {
			err = waitLiveStart(ctx, mlive)
			if err != nil {
				log.Fatal(err)
			}
		} else if err != nil {
			log.Fatal(err)
		}
	} else if twitcasting.Support(url) {
		filename = fmt.Sprintf("twitcasting-%v", time.Now().Local().Format("2006-01-02-15-04"))
		tlive := twitcasting.New(url)
		live = tlive
		err := tlive.WaitStreamStart(ctx, netconn)
		if err != nil {
			log.Fatal(err)
		}
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	ds := make(chan int, 1)
	go func() {
		live.Download(ctx, netconn, fio, filename)
		ds <- 1
	}()

	for {
		select {
		case <-sigchan:
			log.Println("user cancel")
			signal.Reset(os.Interrupt)
			cancel()
			log.Println("wait download loop end")
			<-ds
			return
		case <-ds:
			log.Println("download loop end")
			return
		default:
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
}

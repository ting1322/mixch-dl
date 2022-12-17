package main

import (
	"context"
	"flag"
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
	netconn    inter.INet
	pass       string
)

func parseTime(text string) (time.Time, error) {
	now := time.Now().Local()
	t, err := time.ParseInLocation("15:04", os.Args[2], time.Now().Location())
	if err == nil {
		y, m, d := now.Date()
		t = t.AddDate(y, int(m)-1, d-1)
		if t.Before(now) {
			t = t.AddDate(0, 0, 1)
		}
		return t, nil
	}
	return time.Now(), err
}

func main() {
	flag.StringVar(&pass, "pass", "", "password for twitcasting")
	flag.Parse()
	var url string
	if flag.NArg() > 0 {
		url = flag.Arg(0)
	} else {
		log.Printf(`
need a url as argument, for example:
  mixch-dl https://mixch.tv/u/17209506

  mixch-dl https://mixch.tv/u/17209506/live
    (url with /live is allowed)

  mixch-dl https://mixch.tv/u/17209506  18:57
    (wait until 18:57)

  mixch-dl https://twitcasting.tv/quon01tama
    (twitcasting experimental support, download fmp4 via websocket)

  mixch-dl -pass THE_PASSWORD https://twitcasting.tv/quon01tama
    (twitcasting with password)
`)
		return
	}
	if flag.NArg() > 1 {
		t, err := parseTime(flag.Arg(1))
		if err != nil {
			log.Fatal("time format error", flag.Arg(1), err)
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
		netconn = inter.NewNetConn(url)
		filename = fmt.Sprintf("mixch-%v", time.Now().Local().Format("2006-01-02-15-04"))
		var err error
		live, err = mixch.New(url)
		if err != nil {
			log.Fatal(err)
		}
		err = live.WaitStreamStart(ctx, netconn)
		if err != nil {
			log.Fatal(err)
		}
	} else if twitcasting.Support(url) {
		netconn = inter.NewNetConn(url)
		filename = fmt.Sprintf("twitcasting-%v", time.Now().Local().Format("2006-01-02-15-04"))
		live = twitcasting.New(url, pass)
		err := live.WaitStreamStart(ctx, netconn)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("not support url: %v\n", os.Args[1])
		return
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
			fmt.Printf("\n\nuser cancel\n\n")
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

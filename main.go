package main

import (
	"context"
	"errors"
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
	programVersion string           = "1.x.x-custombuild"
	downloader     *m3u8.Downloader = &m3u8.Downloader{}
	fio            inter.IFs        = &inter.Fs{}
	netconn        inter.INet
	pass           string
	loopAtFinish   bool
)

func parseTime(text string) (time.Time, error) {
	now := time.Now().Local()
	t, err := time.ParseInLocation("15:04", text, time.Now().Location())
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
	var argVersion bool
	flag.StringVar(&pass, "pass", "", "password for twitcasting")
	flag.BoolVar(&loopAtFinish, "loop", false, "continue run even if download finish")
	flag.BoolVar(&argVersion, "version", false, "show program version and exit.")
	flag.Parse()
	if argVersion {
		fmt.Println("mixch-dl", programVersion)
		return
	}
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

	for {
		err := downloadFlow(url)
		if err == inter.ErrNolive {
			// mixch sometimes response HTTP 403 even if live was started.
			// downloadFlow will return ErrNoLive in this case.
			// we need try again, until any video fragment has been download.
			continue
		}
		if err != nil {
			break
		}
		if !loopAtFinish {
			break
		}
	}
}

func downloadFlow(url string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var filename string
	var live inter.Live
	if mixch.Support(url) {
		netconn = inter.NewNetConn(url)
		var err error
		live, err = mixch.New(url)
		if err != nil {
			log.Fatal(err)
		}
		err = live.WaitStreamStart(ctx, netconn)
		if err != nil {
			log.Fatal(err)
		}
		filename = fmt.Sprintf("mixch-%v", time.Now().Local().Format("2006-01-02-15-04"))
	} else if twitcasting.Support(url) {
		netconn = inter.NewNetConn(url)
		live = twitcasting.New(url, pass)
		err := live.WaitStreamStart(ctx, netconn)
		if err != nil {
			log.Fatal(err)
		}
		filename = fmt.Sprintf("twitcasting-%v", time.Now().Local().Format("2006-01-02-15-04"))
	} else {
		fmt.Printf("not support url: %v\n", os.Args[1])
		return errors.New("not support url")
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
			return errors.New("user cancel")
		case <-ds:
			log.Println("download loop end")
			return nil
		default:
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
}

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"mixch-dl/chzzk"
	"mixch-dl/inter"
	"mixch-dl/m3u8"
	"mixch-dl/mixch"
	"mixch-dl/spoon"
	"mixch-dl/twitcasting"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

var (
	programVersion string           = "1.x-dev"
	downloader     *m3u8.Downloader = &m3u8.Downloader{}
	fio            inter.IFs        = &inter.Fs{}
	netconn        inter.INet
	pass           string
	dateFolder     bool
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
	flag.BoolVar(&inter.AutoLoadCookie, "cookies-from-browser", false, "do not load cookie from browser. (default enable)")
	flag.BoolVar(&inter.JsonOutput, "json", false, "output message with json format")
	flag.StringVar(&m3u8.PreferFmt, "prefer-fmt", "", "prefer audio only format")
	flag.BoolVar(&spoon.DownloadChatRoom, "spoon-chat", true, "downlaod spoon chat room (default true, disable by -spoon-chat=false")
	flag.BoolVar(&spoon.EmbedTitle, "spoon-title", true, "add title to mp4 metadata (default true, disable by -spoon-title=false)")
	flag.StringVar(&m3u8.FileExt, "file-ext", ".mp4", "output file extension, default is '.mp4', can change to .m4a")
	flag.BoolVar(&dateFolder, "date-folder", false, "create folder with today date for output file. (default disable)")
	flag.BoolVar(&inter.VerboseOutput, "verbose", false, "output more debug message")
	flag.Parse()
	fmt.Println("mixch-dl", programVersion)
	if argVersion {
		return
	}
	var url string
	if flag.NArg() > 0 {
		url = flag.Arg(0)
	} else {
		fmt.Printf(`
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

  mixch-dl https://www.spooncast.net/jp/live/@lala_ukulele
    (spoon jp)

  mixch-dl -spoon-chat=false https://......
    (spoon, without live chat & htm)
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
		if err != nil && loopAtFinish {
			time.Sleep(30 * time.Second)
		}
		if !loopAtFinish {
			break
		}
		time.Sleep(5 * time.Second)
	}
}

func downloadFlow(url string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var live inter.Live
	inter.LogStatus(inter.STATUS_WaitStream)
	if mixch.Support(url) {
		netconn = inter.NewNetConn(url)
		var err error
		live, err = mixch.New(url)
		if err != nil {
			return err
		}
		err = live.WaitStreamStart(ctx, netconn)
		if err != nil {
			return err
		}
	} else if twitcasting.Support(url) {
		netconn = inter.NewNetConn(url)
		live = twitcasting.New(url, pass)
		err := live.WaitStreamStart(ctx, netconn)
		if err != nil {
			return err
		}
	} else if spoon.Support(url) {
		netconn = inter.NewNetConn(url)
		live = spoon.New(url)
		err := live.WaitStreamStart(ctx, netconn)
		if err != nil {
			return err
		}
	} else if chzzk.Support(url) {
		netconn = inter.NewNetConn(url)
		live = chzzk.New(url)
		err := live.WaitStreamStart(ctx, netconn)
		if err != nil {
			return err
		}
	} else {
		inter.LogMsg(false, fmt.Sprintf("not support url: %v\n", os.Args[1]))
		log.Fatal("not support url")
	}

	var filename string
	filename = time.Now().Local().Format("2006-01-02-15-04")
	if dateFolder {
		var dir = time.Now().Local().Format("2006-01-02")
		if !fio.Exist(dir) {
			os.Mkdir(dir, 0775)
		}

		filename = filepath.Join(dir, filename)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	ds := make(chan int, 1)
	go func() {
		inter.LogStatus(inter.STATUS_Downloading)
		live.Download(ctx, netconn, fio, filename)
		inter.LogStatus(inter.STATUS_Finish)
		ds <- 1
	}()

	for {
		select {
		case <-sigchan:
			inter.LogMsg(false, "user cancel")
			signal.Reset(os.Interrupt)
			cancel()
			inter.LogMsg(false, "wait download loop end")
			<-ds
			return errors.New("user cancel")
		case <-ds:
			inter.LogMsg(false, "download loop end")
			return nil
		default:
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
}

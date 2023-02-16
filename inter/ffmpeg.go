package inter

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func FfmpegMerge(in, out string, fixts bool) {
	log.Printf("merge %v file by ffmpeg\n", out)
	cmdarg := []string{"-i", in, "-c", "copy"}
	if fixts {
		cmdarg = append(cmdarg, "-bsf", "setts=ts=TS-STARTPTS")
	}
	cmdarg = append(cmdarg, "-map", "0", "-dn", "-ignore_unknown")
	if fixts {
		cmdarg = append(cmdarg, "-ss", "1ms")
	}
	cmdarg = append(cmdarg, out)
	var cmd *exec.Cmd = exec.Command("ffmpeg", cmdarg...)
	log.Println(cmd)
	err := cmd.Run()
	if err == nil {
		os.Remove(in)
		//os.Rename(in, in[:len(in)-5])
	}
}

type FfmpegMeta struct {
	Title string
	Artist string
	Comment string
	Album string
}

func FfmpegMetadata(video string, meta FfmpegMeta) {
	outfile := "temp-" + video
	cmdarg := []string{"-i", video}
	if meta.Title != "" {
		cmdarg = append(cmdarg, "-metadata", "title=\"" + meta.Title + "\"")
	}
	if meta.Artist != "" {
		cmdarg = append(cmdarg, "-metadata", "artist=\"" + meta.Artist + "\"")
	}
	if meta.Comment != "" {
		cmdarg = append(cmdarg, "-metadata", "comment=\"" + meta.Comment + "\"")
	}
	if meta.Album != "" {
		cmdarg = append(cmdarg, "-metadata", "album=\"" + meta.Album + "\"")
	}
	cmdarg = append(cmdarg, "-c", "copy", outfile)
	var cmd *exec.Cmd = exec.Command("ffmpeg", cmdarg...)
	log.Println(cmd)
	err := cmd.Run()
	if err != nil {
		log.Println("error at ffmpeg metadata", err)
		return
	}
	replaceFile(outfile, video, video + ".bak")
	if err != nil {
		log.Println("error at ffmpeg metadata", err)
	}
}

func FfmpegFastStartMp4(video string) {
	outfile := "temp-" + video
	cmdarg := []string{"-i", video, "-c", "copy", "-movflags", "+faststart", outfile}
	var cmd *exec.Cmd = exec.Command("ffmpeg", cmdarg...)
	log.Println(cmd)
	err := cmd.Run()
	if err != nil {
		log.Println("error at mp4 faststart", err)
		return
	}
	replaceFile(outfile, video, video + ".bak")
	if err != nil {
		log.Println("error at mp4 faststart", err)
	}
}

func FfmpegAttachThumbnail(video, img string, disposition int) {
	log.Printf("attach cover image to video")
	outfile := "temp-" + video
	cmdarg := []string{"-i", video, "-i", img,
		"-map", "0", "-map", "1",
		fmt.Sprintf("-disposition:%d", disposition), "attached_pic",
		"-c", "copy",
		outfile}
	var cmd *exec.Cmd = exec.Command("ffmpeg", cmdarg...)
	log.Println(cmd)
	err := cmd.Run()
	if err != nil {
		log.Println("error attach thumbnail:", err)
		return
	}
	replaceFile(outfile, video, video + ".bak")
	if err != nil {
		log.Println("error attach thumbnail:", err)
	}
}

func FfprobeTime(filename string) (time.Duration, error) {
	cmdarg := []string{"-v", "error",
		"-show_entries", "format=duration", "-of",
		"default=noprint_wrappers=1:nokey=1", filename}
	var cmd *exec.Cmd = exec.Command("ffprobe", cmdarg...)
	log.Println(cmd)
	outData, err := cmd.Output()
	text := strings.Trim(string(outData), " \r\n")
	log.Println("FFPROBE:", text)
	if err != nil {
		return 0, err
	}
	durationFloat, err := strconv.ParseFloat(text, 64)
	if err != nil {
		log.Println("FFPROBE ERR:", err)
		return 0, err
	}
	floatTime := durationFloat * float64(time.Second)
	return time.Duration(math.Round(floatTime)), nil
}

func replaceFile(src, dest, bak string) error {
	log.Printf("rename %v to %v\n", dest, bak)
	err := os.Rename(dest, bak)
	if err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	log.Printf("rename %v to %v\n", src, dest)
	err = os.Rename(src, dest)
	if err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	log.Printf("rm %v\n", bak)
	os.Remove(bak)
	return nil
}
package inter

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	ffmpegName  string = "ffmpeg"
	ffprobeName string = "ffprobe"
)

func FfmpegMerge(in, out string, fixts bool) {
	LogMsg(false, fmt.Sprintf("merge %v file by ffmpeg", out))
	cmdarg := []string{"-y", "-i", in, "-c", "copy"}
	if fixts {
		// cmdarg = append(cmdarg, "-bsf", "setts=ts=TS-STARTPTS")
		cmdarg = append(cmdarg, "-fps_mode", "auto")
	}
	cmdarg = append(cmdarg, "-map", "0", "-dn", "-ignore_unknown")
	if fixts {
		cmdarg = append(cmdarg, "-ss", "1ms")
	}
	cmdarg = append(cmdarg, out)
	var cmd *exec.Cmd = exec.Command(ffmpegName, cmdarg...)
	LogMsg(false, cmd.String())
	err := cmd.Run()
	if err == nil {
		os.Remove(in)
	}
}

type FfmpegMeta struct {
	Title   string
	Artist  string
	Comment string
	Album   string
}

func FfmpegMetadata(video string, meta FfmpegMeta) {
	outfile := "temp-" + video
	cmdarg := []string{"-y", "-i", video}
	if meta.Title != "" {
		cmdarg = append(cmdarg, "-metadata", "title=\""+meta.Title+"\"")
	}
	if meta.Artist != "" {
		cmdarg = append(cmdarg, "-metadata", "artist=\""+meta.Artist+"\"")
	}
	if meta.Comment != "" {
		cmdarg = append(cmdarg, "-metadata", "comment=\""+meta.Comment+"\"")
	}
	if meta.Album != "" {
		cmdarg = append(cmdarg, "-metadata", "album=\""+meta.Album+"\"")
	}
	cmdarg = append(cmdarg, "-c", "copy", outfile)
	var cmd *exec.Cmd = exec.Command(ffmpegName, cmdarg...)
	LogMsg(false, cmd.String())
	err := cmd.Run()
	if err != nil {
		LogMsg(false, fmt.Sprintf("error at ffmpeg metadata: %v", err))
		return
	}
	replaceFile(outfile, video, video+".bak")
	if err != nil {
		LogMsg(false, fmt.Sprintf("error at ffmpeg metadata: %v", err))
	}
}

func FfmpegFastStartMp4(video string) {
	outfile := "temp-" + video
	cmdarg := []string{"-y", "-i", video, "-c", "copy", "-movflags", "+faststart", outfile}
	var cmd *exec.Cmd = exec.Command(ffmpegName, cmdarg...)
	LogMsg(false, cmd.String())
	err := cmd.Run()
	if err != nil {
		LogMsg(false, fmt.Sprintf("error at mp4 faststart: %v", err))
		return
	}
	replaceFile(outfile, video, video+".bak")
	if err != nil {
		LogMsg(false, fmt.Sprintf("error at mp4 faststart: %v", err))
	}
}

func FfmpegAttachThumbnail(video, img string, disposition int) {
	LogMsg(false, "attach cover image to video")
	outfile := "temp-" + video
	cmdarg := []string{"-y", "-i", video, "-i", img,
		"-map", "0", "-map", "1",
		fmt.Sprintf("-disposition:v:%d", disposition), "attached_pic",
		"-c", "copy",
		outfile}
	var cmd *exec.Cmd = exec.Command(ffmpegName, cmdarg...)
	LogMsg(false, cmd.String())
	err := cmd.Run()
	if err != nil {
		LogMsg(false, fmt.Sprintf("error attach thumbnail: %v", err))
		return
	}
	replaceFile(outfile, video, video+".bak")
	if err != nil {
		LogMsg(false, fmt.Sprintf("error attach thumbnail: %v", err))
	}
}

func FfprobeTime(filename string) (time.Duration, error) {
	cmdarg := []string{"-v", "error",
		"-show_entries", "format=duration", "-of",
		"default=noprint_wrappers=1:nokey=1", filename}
	var cmd *exec.Cmd = exec.Command(ffprobeName, cmdarg...)
	LogMsg(false, cmd.String())
	outData, err := cmd.Output()
	text := strings.Trim(string(outData), " \r\n")
	LogMsg(true, fmt.Sprintf("FFPROBE: %v", text))
	if err != nil {
		return 0, err
	}
	durationFloat, err := strconv.ParseFloat(text, 64)
	if err != nil {
		LogMsg(false, fmt.Sprintf("FFPROBE ERR: %v", err))
		return 0, err
	}
	floatTime := durationFloat * float64(time.Second)
	return time.Duration(math.Round(floatTime)), nil
}

func FfprobeVideoCount(filename string) (int, error) {
	cmdarg := []string{"-v", "error", "-select_streams",
		"v", "-show_entries", "stream=index", "-of",
		"csv=p=0", filename}
	var cmd *exec.Cmd = exec.Command(ffprobeName, cmdarg...)
	LogMsg(false, cmd.String())
	outData, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	text := strings.Trim(string(outData), " \r\n")
	LogMsg(true, fmt.Sprintf("FFPROBE: %v", text))
	if text == "" {
		return 0, nil
	}
	count := len(strings.Split(text, "\n"))
	return count, nil
}

func replaceFile(src, dest, bak string) error {
	LogMsg(false, fmt.Sprintf("replace %v with %v", dest, src))
	LogMsg(true, fmt.Sprintf("rename %v to %v", dest, bak))
	err := os.Rename(dest, bak)
	if err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	LogMsg(true, fmt.Sprintf("rename %v to %v", src, dest))
	err = os.Rename(src, dest)
	if err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	LogMsg(true, fmt.Sprintf("rm %v", bak))
	os.Remove(bak)
	return nil
}

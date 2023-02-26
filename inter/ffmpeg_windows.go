package inter

import (
	"log"
	"os"
	"path/filepath"
)

func init() {
	mypath, _ := os.Executable()
	curDir := filepath.Dir(mypath)
	
	curDirFfmpeg := filepath.Join(curDir, "ffmpeg.exe")
	curDirFfprobe := filepath.Join(curDir, "ffprobe.exe")
	log.Println("try ffmpeg:", curDirFfmpeg)
	if _, err := os.Stat(curDirFfmpeg); os.IsNotExist(err) {
		ffmpegName = "ffmpeg.exe"
	} else {
		ffmpegName = curDirFfmpeg
		log.Println("found ffmpeg in current path")
	}
	if _, err := os.Stat(curDirFfprobe); os.IsNotExist(err) {
		ffprobeName = "ffprobe.exe"
	} else {
		ffprobeName = curDirFfprobe
	}
}

package manager

import (
	"github.com/stashapp/stash/pkg/ffmpeg"
)

func makeScreenshot(probeResult ffmpeg.VideoFile, outputPath string, quality int, width int, time float64) string {
	encoder := ffmpeg.NewEncoder(instance.FFMPEGPath)
	options := ffmpeg.ScreenshotOptions{
		OutputPath: outputPath,
		Quality:    quality,
		Time:       time,
		Width:      width,
	}
	ffmpegerr, _ := encoder.Screenshot(probeResult, options)
	return ffmpegerr
}

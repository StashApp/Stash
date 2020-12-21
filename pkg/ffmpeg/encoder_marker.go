package ffmpeg

import (
	"fmt"
	"strconv"
)

type SceneMarkerOptions struct {
	ScenePath  string
	Seconds    int
	Width      int
	OutputPath string
}

func (e *Encoder) SceneMarkerVideo(probeResult VideoFile, options SceneMarkerOptions) error {
	args := []string{
		"-v", "error",
		"-ss", strconv.Itoa(options.Seconds),
		"-t", "20",
		"-i", probeResult.Path,
		"-max_muxing_queue_size", "1024", // https://trac.ffmpeg.org/ticket/6375
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-profile:v", "high",
		"-level", "4.2",
		"-preset", "veryslow",
		"-crf", "24",
		"-movflags", "+faststart",
		"-threads", "4",
		"-vf", fmt.Sprintf("scale=%v:-2", options.Width),
		"-sws_flags", "lanczos",
		"-c:a", "aac",
		"-b:a", "64k",
		"-strict", "-2",
		options.OutputPath,
	}
	_, _, err := e.run(probeResult, args)
	return err
}

func (e *Encoder) SceneMarkerImage(probeResult VideoFile, options SceneMarkerOptions) error {
	args := []string{
		"-v", "error",
		"-ss", strconv.Itoa(options.Seconds),
		"-t", "5",
		"-i", probeResult.Path,
		"-c:v", "libwebp",
		"-lossless", "1",
		"-q:v", "70",
		"-compression_level", "6",
		"-preset", "default",
		"-loop", "0",
		"-threads", "4",
		"-vf", fmt.Sprintf("scale=%v:-2,fps=12", options.Width),
		"-an",
		options.OutputPath,
	}
	_, _, err := e.run(probeResult, args)
	return err
}

package ffmpeg

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/manager/config"
	"github.com/stashapp/stash/pkg/models"
)

const CopyStreamCodec = "copy"

type Stream struct {
	Stdout   io.ReadCloser
	Process  *os.Process
	options  TranscodeStreamOptions
	mimeType string
}

func (s *Stream) Serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", s.mimeType)
	w.WriteHeader(http.StatusOK)

	logger.Infof("[stream] transcoding video file to %s", s.mimeType)

	// handle if client closes the connection
	notify := r.Context().Done()
	go func() {
		<-notify
		s.Process.Kill()
	}()

	_, err := io.Copy(w, s.Stdout)
	if err != nil {
		logger.Errorf("[stream] error serving transcoded video file: %s", err.Error())
	}
}

type Codec struct {
	Codec     string
	format    string
	MimeType  string
	extraArgs []string
	preArgs   []string
	hls       bool
}

var CodecHLS = Codec{
	Codec:    "libx264",
	format:   "mpegts",
	MimeType: MimeMpegts,
	extraArgs: []string{
		"-acodec", "aac",
		"-pix_fmt", "yuv420p",
		"-preset", "veryfast",
		"-crf", "25",
	},
	hls: true,
}

var CodecH264 = Codec{
	Codec:    "libx264",
	format:   "mp4",
	MimeType: MimeMp4,
	extraArgs: []string{
		"-movflags", "frag_keyframe+empty_moov",
		"-pix_fmt", "yuv420p",
		"-preset", "veryfast",
		"-crf", "25",
	},
}

var CodecVP9 = Codec{
	Codec:    "libvpx-vp9",
	format:   "webm",
	MimeType: MimeWebm,
	extraArgs: []string{
		"-deadline", "realtime",
		"-cpu-used", "5",
		"-row-mt", "1",
		"-crf", "30",
		"-b:v", "0",
	},
}

var CodecVP8 = Codec{
	Codec:    "libvpx",
	format:   "webm",
	MimeType: MimeWebm,
	extraArgs: []string{
		"-deadline", "realtime",
		"-cpu-used", "5",
		"-crf", "12",
		"-b:v", "3M",
		"-pix_fmt", "yuv420p",
	},
}

var CodecHEVC = Codec{
	Codec:    "libx265",
	format:   "mp4",
	MimeType: MimeMp4,
	extraArgs: []string{
		"-movflags", "frag_keyframe",
		"-preset", "veryfast",
		"-crf", "30",
	},
}

// it is very common in MKVs to have just the audio codec unsupported
// copy the video stream, transcode the audio and serve as Matroska
var CodecMKVAudio = Codec{
	Codec:    CopyStreamCodec,
	format:   "matroska",
	MimeType: MimeMkv,
	extraArgs: []string{
		"-c:a", "libopus",
		"-b:a", "96k",
		"-vbr", "on",
	},
}

type TranscodeStreamOptions struct {
	ProbeResult      VideoFile
	Codec            Codec
	StartTime        string
	MaxTranscodeSize models.StreamingResolutionEnum
	// transcode the video, remove the audio
	// in some videos where the audio codec is not supported by ffmpeg
	// ffmpeg fails if you try to transcode the audio
	VideoOnly bool
}

func GetTranscodeStreamOptions(probeResult VideoFile, videoCodec Codec, audioCodec AudioCodec) TranscodeStreamOptions {
	options := TranscodeStreamOptions{
		ProbeResult: probeResult,
		Codec:       videoCodec,
	}

	if audioCodec == MissingUnsupported {
		// ffmpeg fails if it trys to transcode a non supported audio codec
		options.VideoOnly = true
	}

	return options
}

func (o TranscodeStreamOptions) getStreamArgs() []string {
	args := []string{
		"-hide_banner",
		"-v", "error",
	}

	if o.StartTime != "" {
		args = append(args, "-ss", o.StartTime)
	}

	if o.Codec.hls {
		// we only serve a fixed segment length
		args = append(args, "-t", strconv.Itoa(int(hlsSegmentLength)))
	}

	args = append(args,
		"-i", o.ProbeResult.Path,
	)

	if o.VideoOnly {
		args = append(args, "-an")
	}

	args = append(args,
		"-c:v", o.Codec.Codec,
	)

	// don't set scale when copying video stream
	if o.Codec.Codec != CopyStreamCodec {
		scale := calculateTranscodeScale(o.ProbeResult, o.MaxTranscodeSize)
		args = append(args,
			"-vf", "scale="+scale,
		)
	}

	if len(o.Codec.extraArgs) > 0 {
		args = append(args, o.Codec.extraArgs...)
	}

	args = append(args,
		// this is needed for 5-channel ac3 files
		"-ac", "2",
		"-f", o.Codec.format,
		"pipe:",
	)

	return args
}

func (o TranscodeStreamOptions) getHWStreamArgs() []string {
	args := []string{
		"-hide_banner",
		"-v", "error",
	}

	if len(o.Codec.preArgs) > 0 {
		args = append(args, o.Codec.preArgs...)
	}

	if o.StartTime != "" {
		args = append(args, "-ss", o.StartTime)
	}

	if o.Codec.hls {
		// we only serve a fixed segment length
		args = append(args, "-t", strconv.Itoa(int(hlsSegmentLength)))
	}

	args = append(args,
		"-i", o.ProbeResult.Path,
	)

	if len(o.Codec.extraArgs) > 0 {
		args = append(args, o.Codec.extraArgs...)
	}

	args = append(args,
		"-c:v", o.Codec.Codec,
		"-movflags", "frag_keyframe+empty_moov",
		// this is needed for 5-channel ac3 files
		"-ac", "2",
		"-f", o.Codec.format,
		"pipe:",
	)

	return args
}

func (o TranscodeStreamOptions) getHWStreamOptions() [][]string {
	if runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "386") {
		filters := "format=nv12|vaapi,hwupload"

		scaleX, scaleY := calculateHWTranscodeScale(o.ProbeResult, o.MaxTranscodeSize)
		if scaleX != nil && scaleY != nil {
			filters = filters + ",scale_vaapi=w=" + strconv.Itoa(*scaleX) + ":h=" + strconv.Itoa(*scaleY)
		}

		vaapi := Codec{
			Codec:    "h264_vaapi",
			format:   "mp4",
			MimeType: MimeMp4,
			preArgs: []string{
				"-hwaccel", "vaapi",
				"-vaapi_device", "/dev/dri/renderD128",
				"-hwaccel_output_format", "vaapi",
			},
			extraArgs: []string{
				"-vf", filters,
			},
		}

		return [][]string{
			TranscodeStreamOptions{
				ProbeResult: o.ProbeResult,
				StartTime:   o.StartTime,
				Codec:       vaapi,
			}.getHWStreamArgs(),
		}
	} else if runtime.GOOS == "windows" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "386") {
		scale := ""
		scale_qsv := ""

		scaleX, scaleY := calculateHWTranscodeScale(o.ProbeResult, o.MaxTranscodeSize)
		if scaleX != nil && scaleY != nil {
			scale = "scale=" + strconv.Itoa(*scaleX) + ":" + strconv.Itoa(*scaleY)
			scale_qsv = "scale_qsv=" + strconv.Itoa(*scaleX) + ":" + strconv.Itoa(*scaleY)
		}

		vaapi := Codec{
			Codec:    "h264_nvenc",
			format:   "mp4",
			MimeType: MimeMp4,
			preArgs: []string{
				"-hwaccel", "nvdec",
			},
			extraArgs: []string{
				"-preset", "hp",
				"-rc", "vbr_hq",
				"-cq", "30",
				"-qmin:v", "0",
				"-b:v", "0",
				"-pix_fmt", "nv12",
				"-vf", scale,
			},
		}

		qsv := Codec{
			Codec:    "h264_qsv",
			format:   "mp4",
			MimeType: MimeMp4,
			preArgs: []string{
				"-hwaccel", "qsv",
				"-c:v", "h264_qsv",
			},
			extraArgs: []string{
				"-vf", scale_qsv,
				"-look_ahead", "1",
				"-global_quality", "30",
			},
		}

		dxva := Codec{
			Codec:    "h264_qsv",
			format:   "mp4",
			MimeType: MimeMp4,
			preArgs: []string{
				"-hwaccel", "dxva2",
			},
			extraArgs: []string{
				"-vf", scale_qsv,
				"-look_ahead", "1",
				"-global_quality", "30",
			},
		}

		return [][]string{
			TranscodeStreamOptions{
				ProbeResult: o.ProbeResult,
				StartTime:   o.StartTime,
				Codec:       vaapi,
			}.getHWStreamArgs(),
			TranscodeStreamOptions{
				ProbeResult: o.ProbeResult,
				StartTime:   o.StartTime,
				Codec:       qsv,
			}.getHWStreamArgs(),
			TranscodeStreamOptions{
				ProbeResult: o.ProbeResult,
				StartTime:   o.StartTime,
				Codec:       dxva,
			}.getHWStreamArgs(),
		}
	}

	return nil
}

func (e *Encoder) GetTranscodeStream(options TranscodeStreamOptions) (*Stream, error) {
	if config.GetTranscodeHardwareAcceleration() {
		return e.hwStream(options.ProbeResult, options)
	} else {
		return e.stream(options.ProbeResult, options)
	}
}

func (e *Encoder) stream(probeResult VideoFile, options TranscodeStreamOptions) (*Stream, error) {
	args := options.getStreamArgs()
	cmd := exec.Command(e.Path, args...)
	logger.Debugf("Streaming via: %s", strings.Join(cmd.Args, " "))

	stdout, err := cmd.StdoutPipe()
	if nil != err {
		logger.Error("FFMPEG stdout not available: " + err.Error())
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if nil != err {
		logger.Error("FFMPEG stderr not available: " + err.Error())
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	registerRunningEncoder(probeResult.Path, cmd.Process)
	go waitAndDeregister(probeResult.Path, cmd)

	// stderr must be consumed or the process deadlocks
	go func() {
		stderrData, _ := ioutil.ReadAll(stderr)
		stderrString := string(stderrData)
		if len(stderrString) > 0 {
			logger.Debugf("[stream] ffmpeg stderr: %s", stderrString)
		}
	}()

	ret := &Stream{
		Stdout:   stdout,
		Process:  cmd.Process,
		options:  options,
		mimeType: options.Codec.MimeType,
	}
	return ret, nil
}

// Wait 400ms for any stderr output which indicates whether the command was successful
func WaitForError(stderr io.ReadCloser) bool {
	errorChan := make(chan bool)

	go func() {
		ioutil.ReadAll(stderr)
		errorChan <- true
	}()

	select {
	case <-errorChan:
		return true
	case <-time.After(400 * time.Millisecond):
		return false
	}
}

func (e *Encoder) hwStream(probeResult VideoFile, options TranscodeStreamOptions) (*Stream, error) {
	hwEncoders := options.getHWStreamOptions()

	for _, args := range hwEncoders {
		cmd := exec.Command(e.Path, args...)
		logger.Debugf("Streaming via: %s", strings.Join(cmd.Args, " "))

		stdout, err := cmd.StdoutPipe()
		if nil != err {
			logger.Error("FFMPEG stdout not available: " + err.Error())
			return nil, err
		}

		stderr, err := cmd.StderrPipe()
		if nil != err {
			logger.Error("FFMPEG stderr not available: " + err.Error())
			return nil, err
		}

		if err = cmd.Start(); err != nil {
			return nil, err
		}

		hasError := WaitForError(stderr)
		if hasError {
			logger.Error("FFMPEG hardware accelerated transcode failed.")
			continue
		}

		registerRunningEncoder(probeResult.Path, cmd.Process)
		go waitAndDeregister(probeResult.Path, cmd)

		// stderr must be consumed or the process deadlocks
		go func() {
			stderrData, _ := ioutil.ReadAll(stderr)
			stderrString := string(stderrData)
			if len(stderrString) > 0 {
				logger.Debugf("[stream] ffmpeg stderr: %s", stderrString)
			}
		}()

		ret := &Stream{
			Stdout:   stdout,
			Process:  cmd.Process,
			options:  options,
			mimeType: options.Codec.MimeType,
		}
		return ret, nil
	}

	return e.stream(probeResult, options)
}

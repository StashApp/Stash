package manager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

// DestroyScene deletes a scene and its associated relationships from the
// database. Returns a function to perform any post-commit actions.
func DestroyScene(scene *models.Scene, repo models.Repository) (func(), error) {
	qb := repo.Scene()
	mqb := repo.SceneMarker()

	markers, err := mqb.FindBySceneID(scene.ID)
	if err != nil {
		return nil, err
	}

	var funcs []func()
	for _, m := range markers {
		f, err := DestroySceneMarker(scene, m, mqb)
		if err != nil {
			return nil, err
		}
		funcs = append(funcs, f)
	}

	if err := qb.Destroy(scene.ID); err != nil {
		return nil, err
	}

	return func() {
		for _, f := range funcs {
			f()
		}
	}, nil
}

// DestroySceneMarker deletes the scene marker from the database and returns a
// function that removes the generated files, to be executed after the
// transaction is successfully committed.
func DestroySceneMarker(scene *models.Scene, sceneMarker *models.SceneMarker, qb models.SceneMarkerWriter) (func(), error) {
	if err := qb.Destroy(sceneMarker.ID); err != nil {
		return nil, err
	}

	// delete the preview for the marker
	return func() {
		seconds := int(sceneMarker.Seconds)
		DeleteSceneMarkerFiles(scene, seconds, config.GetVideoFileNamingAlgorithm())
	}, nil
}

// DeleteGeneratedSceneFiles deletes generated files for the provided scene.
func DeleteGeneratedSceneFiles(scene *models.Scene, fileNamingAlgo models.HashAlgorithm) {
	sceneHash := scene.GetHash(fileNamingAlgo)

	if sceneHash == "" {
		return
	}

	markersFolder := filepath.Join(GetInstance().Paths.Generated.Markers, sceneHash)

	exists, _ := utils.FileExists(markersFolder)
	if exists {
		err := os.RemoveAll(markersFolder)
		if err != nil {
			logger.Warnf("Could not delete folder %s: %s", markersFolder, err.Error())
		}
	}

	thumbPath := GetInstance().Paths.Scene.GetThumbnailScreenshotPath(sceneHash)
	exists, _ = utils.FileExists(thumbPath)
	if exists {
		err := os.Remove(thumbPath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", thumbPath, err.Error())
		}
	}

	normalPath := GetInstance().Paths.Scene.GetScreenshotPath(sceneHash)
	exists, _ = utils.FileExists(normalPath)
	if exists {
		err := os.Remove(normalPath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", normalPath, err.Error())
		}
	}

	streamPreviewPath := GetInstance().Paths.Scene.GetStreamPreviewPath(sceneHash)
	exists, _ = utils.FileExists(streamPreviewPath)
	if exists {
		err := os.Remove(streamPreviewPath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", streamPreviewPath, err.Error())
		}
	}

	streamPreviewImagePath := GetInstance().Paths.Scene.GetStreamPreviewImagePath(sceneHash)
	exists, _ = utils.FileExists(streamPreviewImagePath)
	if exists {
		err := os.Remove(streamPreviewImagePath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", streamPreviewImagePath, err.Error())
		}
	}

	transcodePath := GetInstance().Paths.Scene.GetTranscodePath(sceneHash)
	exists, _ = utils.FileExists(transcodePath)
	if exists {
		// kill any running streams
		KillRunningStreams(transcodePath)

		err := os.Remove(transcodePath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", transcodePath, err.Error())
		}
	}

	spritePath := GetInstance().Paths.Scene.GetSpriteImageFilePath(sceneHash)
	exists, _ = utils.FileExists(spritePath)
	if exists {
		err := os.Remove(spritePath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", spritePath, err.Error())
		}
	}

	vttPath := GetInstance().Paths.Scene.GetSpriteVttFilePath(sceneHash)
	exists, _ = utils.FileExists(vttPath)
	if exists {
		err := os.Remove(vttPath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", vttPath, err.Error())
		}
	}
}

// DeleteSceneMarkerFiles deletes generated files for a scene marker with the
// provided scene and timestamp.
func DeleteSceneMarkerFiles(scene *models.Scene, seconds int, fileNamingAlgo models.HashAlgorithm) {
	videoPath := GetInstance().Paths.SceneMarkers.GetStreamPath(scene.GetHash(fileNamingAlgo), seconds)
	imagePath := GetInstance().Paths.SceneMarkers.GetStreamPreviewImagePath(scene.GetHash(fileNamingAlgo), seconds)

	exists, _ := utils.FileExists(videoPath)
	if exists {
		err := os.Remove(videoPath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", videoPath, err.Error())
		}
	}

	exists, _ = utils.FileExists(imagePath)
	if exists {
		err := os.Remove(imagePath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", videoPath, err.Error())
		}
	}
}

// DeleteSceneFile deletes the scene video file from the filesystem.
func DeleteSceneFile(scene *models.Scene) {
	// kill any running encoders
	KillRunningStreams(scene.Path)

	err := os.Remove(scene.Path)
	if err != nil {
		logger.Warnf("Could not delete file %s: %s", scene.Path, err.Error())
	}
}

func GetSceneFileContainer(scene *models.Scene) (ffmpeg.Container, error) {
	var container ffmpeg.Container
	if scene.Format.Valid {
		container = ffmpeg.Container(scene.Format.String)
	} else { // container isn't in the DB
		// shouldn't happen, fallback to ffprobe
		tmpVideoFile, err := ffmpeg.NewVideoFile(GetInstance().FFProbePath, scene.Path, false)
		if err != nil {
			return ffmpeg.Container(""), fmt.Errorf("error reading video file: %s", err.Error())
		}

		container = ffmpeg.MatchContainer(tmpVideoFile.Container, scene.Path)
	}

	return container, nil
}

func GetSceneStreamPaths(scene *models.Scene, directStreamURL string) ([]*models.SceneStreamEndpoint, error) {
	if scene == nil {
		return nil, fmt.Errorf("nil scene")
	}

	var ret []*models.SceneStreamEndpoint
	mimeWebm := ffmpeg.MimeWebm
	mimeHLS := ffmpeg.MimeHLS
	mimeMp4 := ffmpeg.MimeMp4

	labelWebm := "webm"
	labelHLS := "HLS"

	// direct stream should only apply when the audio codec is supported
	audioCodec := ffmpeg.MissingUnsupported
	if scene.AudioCodec.Valid {
		audioCodec = ffmpeg.AudioCodec(scene.AudioCodec.String)
	}
	container, err := GetSceneFileContainer(scene)
	if err != nil {
		return nil, err
	}

	if HasTranscode(scene, config.GetVideoFileNamingAlgorithm()) || ffmpeg.IsValidAudioForContainer(audioCodec, container) {
		label := "Direct stream"
		ret = append(ret, &models.SceneStreamEndpoint{
			URL:      directStreamURL,
			MimeType: &mimeMp4,
			Label:    &label,
		})
	}

	// only add mkv stream endpoint if the scene container is an mkv already
	if container == ffmpeg.Matroska {
		label := "mkv"
		ret = append(ret, &models.SceneStreamEndpoint{
			URL: directStreamURL + ".mkv",
			// set mkv to mp4 to trick the client, since many clients won't try mkv
			MimeType: &mimeMp4,
			Label:    &label,
		})
	}

	hls := models.SceneStreamEndpoint{
		URL:      directStreamURL + ".m3u8",
		MimeType: &mimeHLS,
		Label:    &labelHLS,
	}
	ret = append(ret, &hls)

	if !config.GetTranscodeHardwareAcceleration() {
		// WEBM quality transcoding options
		// Note: These have the wrong mime type intentionally to allow jwplayer to selection between mp4/webm
		webmLabelFourK := "WEBM 4K (2160p)"         // "FOUR_K"
		webmLabelFullHD := "WEBM Full HD (1080p)"   // "FULL_HD"
		webmLabelStardardHD := "WEBM HD (720p)"     // "STANDARD_HD"
		webmLabelStandard := "WEBM Standard (480p)" // "STANDARD"
		webmLabelLow := "WEBM Low (240p)"           // "LOW"

		if !scene.Height.Valid || scene.Height.Int64 >= 2160 {
			new := models.SceneStreamEndpoint{
				URL:      directStreamURL + ".webm?resolution=FOUR_K",
				MimeType: &mimeMp4,
				Label:    &webmLabelFourK,
			}
			ret = append(ret, &new)
		}

		if !scene.Height.Valid || scene.Height.Int64 >= 1080 {
			new := models.SceneStreamEndpoint{
				URL:      directStreamURL + ".webm?resolution=FULL_HD",
				MimeType: &mimeMp4,
				Label:    &webmLabelFullHD,
			}
			ret = append(ret, &new)
		}

		if !scene.Height.Valid || scene.Height.Int64 >= 720 {
			new := models.SceneStreamEndpoint{
				URL:      directStreamURL + ".webm?resolution=STANDARD_HD",
				MimeType: &mimeMp4,
				Label:    &webmLabelStardardHD,
			}
			ret = append(ret, &new)
		}

		if !scene.Height.Valid || scene.Height.Int64 >= 480 {
			new := models.SceneStreamEndpoint{
				URL:      directStreamURL + ".webm?resolution=STANDARD",
				MimeType: &mimeMp4,
				Label:    &webmLabelStandard,
			}
			ret = append(ret, &new)
		}

		if !scene.Height.Valid || scene.Height.Int64 >= 240 {
			new := models.SceneStreamEndpoint{
				URL:      directStreamURL + ".webm?resolution=LOW",
				MimeType: &mimeMp4,
				Label:    &webmLabelLow,
			}
			ret = append(ret, &new)
		}
	}

	// Setup up lower quality transcoding options (MP4)
	mp4LabelFourK := "MP4 4K (2160p)"         // "FOUR_K"
	mp4LabelFullHD := "MP4 Full HD (1080p)"   // "FULL_HD"
	mp4LabelStardardHD := "MP4 HD (720p)"     // "STANDARD_HD"
	mp4LabelStandard := "MP4 Standard (480p)" // "STANDARD"
	mp4LabelLow := "MP4 Low (240p)"           // "LOW"

	if !scene.Height.Valid || scene.Height.Int64 >= 2160 {
		new := models.SceneStreamEndpoint{
			URL:      directStreamURL + ".mp4?resolution=FOUR_K",
			MimeType: &mimeMp4,
			Label:    &mp4LabelFourK,
		}
		ret = append(ret, &new)
	}

	if !scene.Height.Valid || scene.Height.Int64 >= 1080 {
		new := models.SceneStreamEndpoint{
			URL:      directStreamURL + ".mp4?resolution=FULL_HD",
			MimeType: &mimeMp4,
			Label:    &mp4LabelFullHD,
		}
		ret = append(ret, &new)
	}

	if !scene.Height.Valid || scene.Height.Int64 >= 720 {
		new := models.SceneStreamEndpoint{
			URL:      directStreamURL + ".mp4?resolution=STANDARD_HD",
			MimeType: &mimeMp4,
			Label:    &mp4LabelStardardHD,
		}
		ret = append(ret, &new)
	}

	if !scene.Height.Valid || scene.Height.Int64 >= 480 {
		new := models.SceneStreamEndpoint{
			URL:      directStreamURL + ".mp4?resolution=STANDARD",
			MimeType: &mimeMp4,
			Label:    &mp4LabelStandard,
		}
		ret = append(ret, &new)
	}

	if !scene.Height.Valid || scene.Height.Int64 >= 240 {
		new := models.SceneStreamEndpoint{
			URL:      directStreamURL + ".mp4?resolution=LOW",
			MimeType: &mimeMp4,
			Label:    &mp4LabelLow,
		}
		ret = append(ret, &new)
	}

	defaultStreams := []*models.SceneStreamEndpoint{
		{
			URL:      directStreamURL + ".webm",
			MimeType: &mimeWebm,
			Label:    &labelWebm,
		},
	}

	ret = append(ret, defaultStreams...)

	return ret, nil
}

// HasTranscode returns true if a transcoded video exists for the provided
// scene. It will check using the OSHash of the scene first, then fall back
// to the checksum.
func HasTranscode(scene *models.Scene, fileNamingAlgo models.HashAlgorithm) bool {
	if scene == nil {
		return false
	}

	sceneHash := scene.GetHash(fileNamingAlgo)
	if sceneHash == "" {
		return false
	}

	transcodePath := instance.Paths.Scene.GetTranscodePath(sceneHash)
	ret, _ := utils.FileExists(transcodePath)
	return ret
}

package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jmoiron/sqlx"

	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

func DestroyScene(sceneID int, tx *sqlx.Tx) error {
	qb := models.NewSceneQueryBuilder()
	jqb := models.NewJoinsQueryBuilder()

	_, err := qb.Find(sceneID)
	if err != nil {
		return err
	}

	if err := jqb.DestroyScenesTags(sceneID, tx); err != nil {
		return err
	}

	if err := jqb.DestroyPerformersScenes(sceneID, tx); err != nil {
		return err
	}

	if err := jqb.DestroyScenesMarkers(sceneID, tx); err != nil {
		return err
	}

	if err := jqb.DestroyScenesGalleries(sceneID, tx); err != nil {
		return err
	}

	if err := qb.Destroy(strconv.Itoa(sceneID), tx); err != nil {
		return err
	}

	return nil
}

func DeleteGeneratedSceneFiles(scene *models.Scene) {
	markersFolder := filepath.Join(GetInstance().Paths.Generated.Markers, scene.Checksum)

	exists, _ := utils.FileExists(markersFolder)
	if exists {
		err := os.RemoveAll(markersFolder)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", scene.Path, err.Error())
		}
	}

	thumbPath := GetInstance().Paths.Scene.GetThumbnailScreenshotPath(scene.Checksum)
	exists, _ = utils.FileExists(thumbPath)
	if exists {
		err := os.Remove(thumbPath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", thumbPath, err.Error())
		}
	}

	normalPath := GetInstance().Paths.Scene.GetScreenshotPath(scene.Checksum)
	exists, _ = utils.FileExists(normalPath)
	if exists {
		err := os.Remove(normalPath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", normalPath, err.Error())
		}
	}

	streamPreviewPath := GetInstance().Paths.Scene.GetStreamPreviewPath(scene.Checksum)
	exists, _ = utils.FileExists(streamPreviewPath)
	if exists {
		err := os.Remove(streamPreviewPath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", streamPreviewPath, err.Error())
		}
	}

	streamPreviewImagePath := GetInstance().Paths.Scene.GetStreamPreviewImagePath(scene.Checksum)
	exists, _ = utils.FileExists(streamPreviewImagePath)
	if exists {
		err := os.Remove(streamPreviewImagePath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", streamPreviewImagePath, err.Error())
		}
	}

	transcodePath := GetInstance().Paths.Scene.GetTranscodePath(scene.Checksum)
	exists, _ = utils.FileExists(transcodePath)
	if exists {
		// kill any running streams
		KillRunningStreams(transcodePath)

		err := os.Remove(transcodePath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", transcodePath, err.Error())
		}
	}

	spritePath := GetInstance().Paths.Scene.GetSpriteImageFilePath(scene.Checksum)
	exists, _ = utils.FileExists(spritePath)
	if exists {
		err := os.Remove(spritePath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", spritePath, err.Error())
		}
	}

	vttPath := GetInstance().Paths.Scene.GetSpriteVttFilePath(scene.Checksum)
	exists, _ = utils.FileExists(vttPath)
	if exists {
		err := os.Remove(vttPath)
		if err != nil {
			logger.Warnf("Could not delete file %s: %s", vttPath, err.Error())
		}
	}
}

func DeleteSceneMarkerFiles(scene *models.Scene, seconds int) {
	videoPath := GetInstance().Paths.SceneMarkers.GetStreamPath(scene.Checksum, seconds)
	imagePath := GetInstance().Paths.SceneMarkers.GetStreamPreviewImagePath(scene.Checksum, seconds)

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
		tmpVideoFile, err := ffmpeg.NewVideoFile(GetInstance().FFProbePath, scene.Path)
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
	labelMp4 := "mp4"

	// direct stream should only apply when the audio codec is supported
	audioCodec := ffmpeg.MissingUnsupported
	if scene.AudioCodec.Valid {
		audioCodec = ffmpeg.AudioCodec(scene.AudioCodec.String)
	}
	container, err := GetSceneFileContainer(scene)
	if err != nil {
		return nil, err
	}

	hasTranscode, _ := HasTranscode(scene)
	if hasTranscode || ffmpeg.IsValidAudioForContainer(audioCodec, container) {
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

	defaultStreams := []*models.SceneStreamEndpoint{
		{
			URL:      directStreamURL + ".webm",
			MimeType: &mimeWebm,
			Label:    &labelWebm,
		},
		{
			URL:      directStreamURL + ".m3u8",
			MimeType: &mimeHLS,
			Label:    &labelHLS,
		},
		{
			URL:      directStreamURL + ".mp4",
			MimeType: &mimeMp4,
			Label:    &labelMp4,
		},
	}

	ret = append(ret, defaultStreams...)

	// TODO - at some point, look at streaming at various resolutions

	return ret, nil
}

func HasTranscode(scene *models.Scene) (bool, error) {
	if scene == nil {
		return false, fmt.Errorf("nil scene")
	}
	transcodePath := instance.Paths.Scene.GetTranscodePath(scene.Checksum)
	return utils.FileExists(transcodePath)
}

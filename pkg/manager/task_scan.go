package manager

import (
	"archive/zip"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/remeh/sizedwaitgroup"

	"github.com/stashapp/stash/pkg/database"
	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/image"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

type ScanTask struct {
	FilePath             string
	UseFileMetadata      bool
	calculateMD5         bool
	fileNamingAlgorithm  models.HashAlgorithm
	GenerateSprite       bool
	GeneratePreview      bool
	GenerateImagePreview bool
	zipGallery           *models.Gallery
}

func (t *ScanTask) Start(wg *sizedwaitgroup.SizedWaitGroup) {
	if isGallery(t.FilePath) {
		t.scanGallery()
	} else if isVideo(t.FilePath) {
		scene := t.scanScene()

		if scene != nil {
			iwg := sizedwaitgroup.New(2)

			if t.GenerateSprite {
				iwg.Add()
				taskSprite := GenerateSpriteTask{Scene: *scene, Overwrite: false, fileNamingAlgorithm: t.fileNamingAlgorithm}
				go taskSprite.Start(&iwg)
			}

			if t.GeneratePreview {
				iwg.Add()

				var previewSegmentDuration = config.GetPreviewSegmentDuration()
				var previewSegments = config.GetPreviewSegments()
				var previewExcludeStart = config.GetPreviewExcludeStart()
				var previewExcludeEnd = config.GetPreviewExcludeEnd()
				var previewPresent = config.GetPreviewPreset()

				// NOTE: the reuse of this model like this is painful.
				previewOptions := models.GeneratePreviewOptionsInput{
					PreviewSegments:        &previewSegments,
					PreviewSegmentDuration: &previewSegmentDuration,
					PreviewExcludeStart:    &previewExcludeStart,
					PreviewExcludeEnd:      &previewExcludeEnd,
					PreviewPreset:          &previewPresent,
				}

				taskPreview := GeneratePreviewTask{
					Scene:               *scene,
					ImagePreview:        t.GenerateImagePreview,
					Options:             previewOptions,
					Overwrite:           false,
					fileNamingAlgorithm: t.fileNamingAlgorithm,
				}
				go taskPreview.Start(&iwg)
			}

			iwg.Wait()
		}
	} else if isImage(t.FilePath) {
		t.scanImage()
	}

	wg.Done()
}

func (t *ScanTask) scanGallery() {
	qb := models.NewGalleryQueryBuilder()
	gallery, _ := qb.FindByPath(t.FilePath)

	fileModTime, err := t.getFileModTime()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	if gallery != nil {
		// We already have this item in the database, keep going

		// if file mod time is not set, set it now
		// we will also need to rescan the zip contents
		updateModTime := false
		if !gallery.FileModTime.Valid {
			updateModTime = true
			t.updateFileModTime(gallery.ID, fileModTime, &qb)

			// update our copy of the gallery
			var err error
			gallery, err = qb.Find(gallery.ID, nil)
			if err != nil {
				logger.Error(err.Error())
				return
			}
		}

		// if the mod time of the zip file is different than that of the associated
		// gallery, then recalculate the checksum
		modified := t.isFileModified(fileModTime, gallery.FileModTime)
		if modified {
			logger.Infof("%s has been updated: rescanning", t.FilePath)

			// update the checksum and the modification time
			checksum, err := t.calculateChecksum()
			if err != nil {
				logger.Error(err.Error())
				return
			}

			currentTime := time.Now()
			galleryPartial := models.GalleryPartial{
				ID:       gallery.ID,
				Checksum: &checksum,
				FileModTime: &models.NullSQLiteTimestamp{
					Timestamp: fileModTime,
					Valid:     true,
				},
				UpdatedAt: &models.SQLiteTimestamp{Timestamp: currentTime},
			}

			err = database.WithTxn(func(tx *sqlx.Tx) error {
				_, err := qb.UpdatePartial(galleryPartial, tx)
				return err
			})
			if err != nil {
				logger.Error(err.Error())
				return
			}
		}

		// scan the zip files if the gallery has no images
		iqb := models.NewImageQueryBuilder()
		images, err := iqb.CountByGalleryID(gallery.ID)
		if err != nil {
			logger.Errorf("error getting images for zip gallery %s: %s", t.FilePath, err.Error())
		}

		if images == 0 || modified || updateModTime {
			t.scanZipImages(gallery)
		} else {
			// in case thumbnails have been deleted, regenerate them
			t.regenerateZipImages(gallery)
		}
		return
	}

	// Ignore directories.
	if isDir, _ := utils.DirExists(t.FilePath); isDir {
		return
	}

	checksum, err := t.calculateChecksum()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	ctx := context.TODO()
	tx := database.DB.MustBeginTx(ctx, nil)
	gallery, _ = qb.FindByChecksum(checksum, tx)
	if gallery != nil {
		exists, _ := utils.FileExists(gallery.Path.String)
		if exists {
			logger.Infof("%s already exists.  Duplicate of %s ", t.FilePath, gallery.Path.String)
		} else {
			logger.Infof("%s already exists.  Updating path...", t.FilePath)
			gallery.Path = sql.NullString{
				String: t.FilePath,
				Valid:  true,
			}
			gallery, err = qb.Update(*gallery, tx)
		}
	} else {
		currentTime := time.Now()

		newGallery := models.Gallery{
			Checksum: checksum,
			Zip:      true,
			Path: sql.NullString{
				String: t.FilePath,
				Valid:  true,
			},
			FileModTime: models.NullSQLiteTimestamp{
				Timestamp: fileModTime,
				Valid:     true,
			},
			CreatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
			UpdatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
		}

		// don't create gallery if it has no images
		if countImagesInZip(t.FilePath) > 0 {
			// only warn when creating the gallery
			ok, err := utils.IsZipFileUncompressed(t.FilePath)
			if err == nil && !ok {
				logger.Warnf("%s is using above store (0) level compression.", t.FilePath)
			}

			logger.Infof("%s doesn't exist.  Creating new item...", t.FilePath)
			gallery, err = qb.Create(newGallery, tx)
		}
	}

	if err != nil {
		logger.Error(err.Error())
		tx.Rollback()
		return
	}

	err = tx.Commit()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	// if the gallery has no associated images, then scan the zip for images
	if gallery != nil {
		t.scanZipImages(gallery)
	}
}

type fileModTimeUpdater interface {
	UpdateFileModTime(id int, modTime models.NullSQLiteTimestamp, tx *sqlx.Tx) error
}

func (t *ScanTask) updateFileModTime(id int, fileModTime time.Time, updater fileModTimeUpdater) error {
	logger.Infof("setting file modification time on %s", t.FilePath)

	err := database.WithTxn(func(tx *sqlx.Tx) error {
		return updater.UpdateFileModTime(id, models.NullSQLiteTimestamp{
			Timestamp: fileModTime,
			Valid:     true,
		}, tx)
	})

	if err != nil {
		return err
	}

	return nil
}

func (t *ScanTask) getFileModTime() (time.Time, error) {
	fi, err := os.Stat(t.FilePath)
	if err != nil {
		return time.Time{}, fmt.Errorf("error performing stat on %s: %s", t.FilePath, err.Error())
	}

	ret := fi.ModTime()
	// truncate to seconds, since we don't store beyond that in the database
	ret = ret.Truncate(time.Second)

	return ret, nil
}

func (t *ScanTask) isFileModified(fileModTime time.Time, modTime models.NullSQLiteTimestamp) bool {
	return !modTime.Timestamp.Equal(fileModTime)
}

// associates a gallery to a scene with the same basename
func (t *ScanTask) associateGallery(wg *sizedwaitgroup.SizedWaitGroup) {
	qb := models.NewGalleryQueryBuilder()
	gallery, _ := qb.FindByPath(t.FilePath)
	if gallery == nil {
		// associate is run after scan is finished
		// should only happen if gallery is a directory or an io error occurs during hashing
		logger.Warnf("associate: gallery %s not found in DB", t.FilePath)
		wg.Done()
		return
	}

	// gallery has no SceneID
	if !gallery.SceneID.Valid {
		basename := strings.TrimSuffix(t.FilePath, filepath.Ext(t.FilePath))
		var relatedFiles []string
		vExt := config.GetVideoExtensions()
		// make a list of media files that can be related to the gallery
		for _, ext := range vExt {
			related := basename + "." + ext
			// exclude gallery extensions from the related files
			if !isGallery(related) {
				relatedFiles = append(relatedFiles, related)
			}
		}
		for _, scenePath := range relatedFiles {
			qbScene := models.NewSceneQueryBuilder()
			scene, _ := qbScene.FindByPath(scenePath)
			// found related Scene
			if scene != nil {
				logger.Infof("associate: Gallery %s is related to scene: %d", t.FilePath, scene.ID)

				gallery.SceneID.Int64 = int64(scene.ID)
				gallery.SceneID.Valid = true

				ctx := context.TODO()
				tx := database.DB.MustBeginTx(ctx, nil)

				_, err := qb.Update(*gallery, tx)
				if err != nil {
					logger.Errorf("associate: Error updating gallery sceneId %s", err)
					_ = tx.Rollback()
				} else if err := tx.Commit(); err != nil {
					logger.Error(err.Error())
				}

				// since a gallery can have only one related scene
				// only first found is associated
				break
			}
		}
	}
	wg.Done()
}

func (t *ScanTask) scanScene() *models.Scene {
	qb := models.NewSceneQueryBuilder()
	scene, _ := qb.FindByPath(t.FilePath)

	fileModTime, err := t.getFileModTime()
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	if scene != nil {
		// if file mod time is not set, set it now
		if !scene.FileModTime.Valid {
			t.updateFileModTime(scene.ID, fileModTime, &qb)

			// update our copy of the scene
			var err error
			scene, err = qb.Find(scene.ID)
			if err != nil {
				logger.Error(err.Error())
				return nil
			}
		}

		// if the mod time of the file is different than that of the associated
		// scene, then recalculate the checksum and regenerate the thumbnail
		modified := t.isFileModified(fileModTime, scene.FileModTime)
		if modified {
			scene, err = t.rescanScene(scene, fileModTime)
			if err != nil {
				logger.Error(err.Error())
				return nil
			}
		}

		// We already have this item in the database
		// check for thumbnails,screenshots
		t.makeScreenshots(nil, scene.GetHash(t.fileNamingAlgorithm), scene.ID)

		// check for container
		if !scene.Format.Valid {
			videoFile, err := ffmpeg.NewVideoFile(instance.FFProbePath, t.FilePath)
			if err != nil {
				logger.Error(err.Error())
				models.PushSceneError(scene.ID, "ffprobe", "scan", err.Error())
				return nil
			}
			container := ffmpeg.MatchContainer(videoFile.Container, t.FilePath)
			logger.Infof("Adding container %s to file %s", container, t.FilePath)

			ctx := context.TODO()
			tx := database.DB.MustBeginTx(ctx, nil)
			err = qb.UpdateFormat(scene.ID, string(container), tx)
			if err != nil {
				logger.Error(err.Error())
				_ = tx.Rollback()
			} else if err := tx.Commit(); err != nil {
				logger.Error(err.Error())
			}
		}

		// check if oshash is set
		if !scene.OSHash.Valid {
			logger.Infof("Calculating oshash for existing file %s ...", t.FilePath)
			oshash, err := utils.OSHashFromFilePath(t.FilePath)
			if err != nil {
				logger.Error(err.Error())
				return nil
			}

			// check if oshash clashes with existing scene
			dupe, _ := qb.FindByOSHash(oshash)
			if dupe != nil {
				logger.Errorf("OSHash for file %s is the same as that of %s", t.FilePath, dupe.Path)
				return nil
			}

			ctx := context.TODO()
			tx := database.DB.MustBeginTx(ctx, nil)
			err = qb.UpdateOSHash(scene.ID, oshash, tx)
			if err != nil {
				logger.Error(err.Error())
				tx.Rollback()
				return nil
			} else if err := tx.Commit(); err != nil {
				logger.Error(err.Error())
			}
		}

		// check if MD5 is set, if calculateMD5 is true
		if t.calculateMD5 && !scene.Checksum.Valid {
			checksum, err := t.calculateChecksum()
			if err != nil {
				logger.Error(err.Error())
				return nil
			}

			// check if checksum clashes with existing scene
			dupe, _ := qb.FindByChecksum(checksum)
			if dupe != nil {
				logger.Errorf("MD5 for file %s is the same as that of %s", t.FilePath, dupe.Path)
				return nil
			}

			ctx := context.TODO()
			tx := database.DB.MustBeginTx(ctx, nil)
			err = qb.UpdateChecksum(scene.ID, checksum, tx)
			if err != nil {
				logger.Error(err.Error())
				_ = tx.Rollback()
			} else if err := tx.Commit(); err != nil {
				logger.Error(err.Error())
			}
		}

		return nil
	}

	// Ignore directories.
	if isDir, _ := utils.DirExists(t.FilePath); isDir {
		return nil
	}

	videoFile, err := ffmpeg.NewVideoFile(instance.FFProbePath, t.FilePath)
	if err != nil {
		logger.Error(err.Error())
		models.PushSceneError(-1, "ffprobe", "scan", err.Error())
		return nil
	}
	container := ffmpeg.MatchContainer(videoFile.Container, t.FilePath)

	// Override title to be filename if UseFileMetadata is false
	if !t.UseFileMetadata {
		videoFile.SetTitleFromPath()
	}

	var checksum string

	logger.Infof("%s not found. Calculating oshash...", t.FilePath)
	oshash, err := utils.OSHashFromFilePath(t.FilePath)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	if t.fileNamingAlgorithm == models.HashAlgorithmMd5 || t.calculateMD5 {
		checksum, err = t.calculateChecksum()
		if err != nil {
			logger.Error(err.Error())
			return nil
		}
	}

	// check for scene by checksum and oshash - MD5 should be
	// redundant, but check both
	if checksum != "" {
		scene, _ = qb.FindByChecksum(checksum)
	}

	if scene == nil {
		scene, _ = qb.FindByOSHash(oshash)
	}

	sceneHash := oshash

	if t.fileNamingAlgorithm == models.HashAlgorithmMd5 {
		sceneHash = checksum
	}

	var retScene *models.Scene

	ctx := context.TODO()
	tx := database.DB.MustBeginTx(ctx, nil)
	if scene != nil {
		exists, _ := utils.FileExists(scene.Path)
		if exists {
			logger.Infof("%s already exists. Duplicate of %s", t.FilePath, scene.Path)
			models.PushSceneError(scene.ID, "Duplicate", "scan", t.FilePath)
		} else {
			logger.Infof("%s already exists. Updating path...", t.FilePath)
			scenePartial := models.ScenePartial{
				ID:   scene.ID,
				Path: &t.FilePath,
			}
			_, err = qb.Update(scenePartial, tx)
		}
	} else {
		logger.Infof("%s doesn't exist. Creating new item...", t.FilePath)
		currentTime := time.Now()
		newScene := models.Scene{
			Checksum:   sql.NullString{String: checksum, Valid: checksum != ""},
			OSHash:     sql.NullString{String: oshash, Valid: oshash != ""},
			Path:       t.FilePath,
			Title:      sql.NullString{String: videoFile.Title, Valid: true},
			Duration:   sql.NullFloat64{Float64: videoFile.Duration, Valid: true},
			VideoCodec: sql.NullString{String: videoFile.VideoCodec, Valid: true},
			AudioCodec: sql.NullString{String: videoFile.AudioCodec, Valid: true},
			Format:     sql.NullString{String: string(container), Valid: true},
			Width:      sql.NullInt64{Int64: int64(videoFile.Width), Valid: true},
			Height:     sql.NullInt64{Int64: int64(videoFile.Height), Valid: true},
			Framerate:  sql.NullFloat64{Float64: videoFile.FrameRate, Valid: true},
			Bitrate:    sql.NullInt64{Int64: videoFile.Bitrate, Valid: true},
			Size:       sql.NullString{String: strconv.Itoa(int(videoFile.Size)), Valid: true},
			FileModTime: models.NullSQLiteTimestamp{
				Timestamp: fileModTime,
				Valid:     true,
			},
			CreatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
			UpdatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
		}

		if t.UseFileMetadata {
			newScene.Details = sql.NullString{String: videoFile.Comment, Valid: true}
			newScene.Date = models.SQLiteDate{String: videoFile.CreationTime.Format("2006-01-02")}
		}

		retScene, err = qb.Create(newScene, tx)
	}

	if retScene != nil {
		t.makeScreenshots(videoFile, sceneHash, retScene.ID)
	}

	if err != nil {
		logger.Error(err.Error())
		_ = tx.Rollback()
		return nil

	} else if err := tx.Commit(); err != nil {
		logger.Error(err.Error())
		return nil
	}

	return retScene
}

func (t *ScanTask) rescanScene(scene *models.Scene, fileModTime time.Time) (*models.Scene, error) {
	logger.Infof("%s has been updated: rescanning", t.FilePath)

	// update the oshash/checksum and the modification time
	logger.Infof("Calculating oshash for existing file %s ...", t.FilePath)
	oshash, err := utils.OSHashFromFilePath(t.FilePath)
	if err != nil {
		return nil, err
	}

	var checksum *sql.NullString
	if t.calculateMD5 {
		cs, err := t.calculateChecksum()
		if err != nil {
			return nil, err
		}

		checksum = &sql.NullString{
			String: cs,
			Valid:  true,
		}
	}

	// regenerate the file details as well
	videoFile, err := ffmpeg.NewVideoFile(instance.FFProbePath, t.FilePath)
	if err != nil {
		return nil, err
	}
	container := ffmpeg.MatchContainer(videoFile.Container, t.FilePath)

	currentTime := time.Now()
	scenePartial := models.ScenePartial{
		ID:       scene.ID,
		Checksum: checksum,
		OSHash: &sql.NullString{
			String: oshash,
			Valid:  true,
		},
		Duration:   &sql.NullFloat64{Float64: videoFile.Duration, Valid: true},
		VideoCodec: &sql.NullString{String: videoFile.VideoCodec, Valid: true},
		AudioCodec: &sql.NullString{String: videoFile.AudioCodec, Valid: true},
		Format:     &sql.NullString{String: string(container), Valid: true},
		Width:      &sql.NullInt64{Int64: int64(videoFile.Width), Valid: true},
		Height:     &sql.NullInt64{Int64: int64(videoFile.Height), Valid: true},
		Framerate:  &sql.NullFloat64{Float64: videoFile.FrameRate, Valid: true},
		Bitrate:    &sql.NullInt64{Int64: videoFile.Bitrate, Valid: true},
		Size:       &sql.NullString{String: strconv.Itoa(int(videoFile.Size)), Valid: true},
		FileModTime: &models.NullSQLiteTimestamp{
			Timestamp: fileModTime,
			Valid:     true,
		},
		UpdatedAt: &models.SQLiteTimestamp{Timestamp: currentTime},
	}

	var ret *models.Scene
	err = database.WithTxn(func(tx *sqlx.Tx) error {
		qb := models.NewSceneQueryBuilder()
		var txnErr error
		ret, txnErr = qb.Update(scenePartial, tx)
		return txnErr
	})
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	// leave the generated files as is - the scene file may have been moved
	// elsewhere

	return ret, nil
}

func (t *ScanTask) makeScreenshots(probeResult *ffmpeg.VideoFile, checksum string, sceneID int) {
	thumbPath := instance.Paths.Scene.GetThumbnailScreenshotPath(checksum)
	normalPath := instance.Paths.Scene.GetScreenshotPath(checksum)

	thumbExists, _ := utils.FileExists(thumbPath)
	normalExists, _ := utils.FileExists(normalPath)

	if thumbExists && normalExists {
		return
	}

	if probeResult == nil {
		var err error
		probeResult, err = ffmpeg.NewVideoFile(instance.FFProbePath, t.FilePath)

		if err != nil {
			logger.Error(err.Error())
			models.PushSceneError(sceneID, "ffprobe", "scan", err.Error())
			return
		}
		logger.Infof("Regenerating images for %s", t.FilePath)
	}

	at := float64(probeResult.Duration) * 0.2

	if !thumbExists {
		logger.Debugf("Creating thumbnail for %s", t.FilePath)
		ffmpegerr := makeScreenshot(*probeResult, thumbPath, 5, 320, at)
		if ffmpegerr != "" {
			models.PushSceneError(sceneID, "ffmpeg", "scan", ffmpegerr)
		}
	}

	if !normalExists {
		logger.Debugf("Creating screenshot for %s", t.FilePath)
		ffmpegerr := makeScreenshot(*probeResult, normalPath, 2, probeResult.Width, at)
		if ffmpegerr != "" {
			models.PushSceneError(sceneID, "ffmpeg", "scan", ffmpegerr)
		}
	}
}

func (t *ScanTask) scanZipImages(zipGallery *models.Gallery) {
	err := walkGalleryZip(zipGallery.Path.String, func(file *zip.File) error {
		// copy this task and change the filename
		subTask := *t

		// filepath is the zip file and the internal file name, separated by a null byte
		subTask.FilePath = image.ZipFilename(zipGallery.Path.String, file.Name)
		subTask.zipGallery = zipGallery

		// run the subtask and wait for it to complete
		iwg := sizedwaitgroup.New(1)
		iwg.Add()
		subTask.Start(&iwg)
		return nil
	})
	if err != nil {
		logger.Warnf("failed to scan zip file images for %s: %s", zipGallery.Path.String, err.Error())
	}
}

func (t *ScanTask) regenerateZipImages(zipGallery *models.Gallery) {
	iqb := models.NewImageQueryBuilder()

	images, err := iqb.FindByGalleryID(zipGallery.ID)
	if err != nil {
		logger.Warnf("failed to find gallery images: %s", err.Error())
		return
	}

	for _, img := range images {
		t.generateThumbnail(img)
	}
}

func (t *ScanTask) scanImage() {
	qb := models.NewImageQueryBuilder()
	i, _ := qb.FindByPath(t.FilePath)

	fileModTime, err := image.GetFileModTime(t.FilePath)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	if i != nil {
		// if file mod time is not set, set it now
		if !i.FileModTime.Valid {
			t.updateFileModTime(i.ID, fileModTime, &qb)

			// update our copy of the gallery
			var err error
			i, err = qb.Find(i.ID)
			if err != nil {
				logger.Error(err.Error())
				return
			}
		}

		// if the mod time of the file is different than that of the associated
		// image, then recalculate the checksum and regenerate the thumbnail
		modified := t.isFileModified(fileModTime, i.FileModTime)
		if modified {
			i, err = t.rescanImage(i, fileModTime)
			if err != nil {
				logger.Error(err.Error())
				return
			}
		}

		// We already have this item in the database
		// check for thumbnails
		t.generateThumbnail(i)

		return
	}

	// Ignore directories.
	if isDir, _ := utils.DirExists(t.FilePath); isDir {
		return
	}

	var checksum string

	logger.Infof("%s not found.  Calculating checksum...", t.FilePath)
	checksum, err = t.calculateImageChecksum()
	if err != nil {
		logger.Errorf("error calculating checksum for %s: %s", t.FilePath, err.Error())
		return
	}

	// check for scene by checksum and oshash - MD5 should be
	// redundant, but check both
	i, _ = qb.FindByChecksum(checksum)

	ctx := context.TODO()
	tx := database.DB.MustBeginTx(ctx, nil)
	if i != nil {
		exists := image.FileExists(i.Path)
		if exists {
			logger.Infof("%s already exists.  Duplicate of %s ", image.PathDisplayName(t.FilePath), image.PathDisplayName(i.Path))
		} else {
			logger.Infof("%s already exists.  Updating path...", image.PathDisplayName(t.FilePath))
			imagePartial := models.ImagePartial{
				ID:   i.ID,
				Path: &t.FilePath,
			}
			_, err = qb.Update(imagePartial, tx)
		}
	} else {
		logger.Infof("%s doesn't exist.  Creating new item...", image.PathDisplayName(t.FilePath))
		currentTime := time.Now()
		newImage := models.Image{
			Checksum: checksum,
			Path:     t.FilePath,
			FileModTime: models.NullSQLiteTimestamp{
				Timestamp: fileModTime,
				Valid:     true,
			},
			CreatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
			UpdatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
		}
		err = image.SetFileDetails(&newImage)
		if err == nil {
			i, err = qb.Create(newImage, tx)
		}
	}

	if err == nil {
		jqb := models.NewJoinsQueryBuilder()
		if t.zipGallery != nil {
			// associate with gallery
			_, err = jqb.AddImageGallery(i.ID, t.zipGallery.ID, tx)
		} else if config.GetCreateGalleriesFromFolders() {
			// create gallery from folder or associate with existing gallery
			logger.Infof("Associating image %s with folder gallery", i.Path)
			err = t.associateImageWithFolderGallery(i.ID, tx)
		}
	}

	if err != nil {
		logger.Error(err.Error())
		_ = tx.Rollback()
		return
	} else if err := tx.Commit(); err != nil {
		logger.Error(err.Error())
		return
	}

	t.generateThumbnail(i)
}

func (t *ScanTask) rescanImage(i *models.Image, fileModTime time.Time) (*models.Image, error) {
	logger.Infof("%s has been updated: rescanning", t.FilePath)

	oldChecksum := i.Checksum

	// update the checksum and the modification time
	checksum, err := t.calculateImageChecksum()
	if err != nil {
		return nil, err
	}

	// regenerate the file details as well
	fileDetails, err := image.GetFileDetails(t.FilePath)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	imagePartial := models.ImagePartial{
		ID:       i.ID,
		Checksum: &checksum,
		Width:    &fileDetails.Width,
		Height:   &fileDetails.Height,
		Size:     &fileDetails.Size,
		FileModTime: &models.NullSQLiteTimestamp{
			Timestamp: fileModTime,
			Valid:     true,
		},
		UpdatedAt: &models.SQLiteTimestamp{Timestamp: currentTime},
	}

	var ret *models.Image
	err = database.WithTxn(func(tx *sqlx.Tx) error {
		qb := models.NewImageQueryBuilder()
		var txnErr error
		ret, txnErr = qb.Update(imagePartial, tx)
		return txnErr
	})
	if err != nil {
		return nil, err
	}

	// remove the old thumbnail if the checksum changed - we'll regenerate it
	if oldChecksum != checksum {
		err = os.Remove(GetInstance().Paths.Generated.GetThumbnailPath(oldChecksum, models.DefaultGthumbWidth)) // remove cache dir of gallery
		if err != nil {
			logger.Errorf("Error deleting thumbnail image: %s", err)
		}
	}

	return ret, nil
}

func (t *ScanTask) associateImageWithFolderGallery(imageID int, tx *sqlx.Tx) error {
	// find a gallery with the path specified
	path := filepath.Dir(t.FilePath)
	gqb := models.NewGalleryQueryBuilder()
	jqb := models.NewJoinsQueryBuilder()
	g, err := gqb.FindByPath(path)
	if err != nil {
		return err
	}

	if g == nil {
		checksum := utils.MD5FromString(path)

		// create the gallery
		currentTime := time.Now()

		newGallery := models.Gallery{
			Checksum: checksum,
			Path: sql.NullString{
				String: path,
				Valid:  true,
			},
			CreatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
			UpdatedAt: models.SQLiteTimestamp{Timestamp: currentTime},
		}

		logger.Infof("Creating gallery for folder %s", path)
		g, err = gqb.Create(newGallery, tx)
		if err != nil {
			return err
		}
	}

	// associate image with gallery
	_, err = jqb.AddImageGallery(imageID, g.ID, tx)
	return err
}

func (t *ScanTask) generateThumbnail(i *models.Image) {
	thumbPath := GetInstance().Paths.Generated.GetThumbnailPath(i.Checksum, models.DefaultGthumbWidth)
	exists, _ := utils.FileExists(thumbPath)
	if exists {
		return
	}

	srcImage, err := image.GetSourceImage(i)
	if err != nil {
		logger.Errorf("error reading image %s: %s", i.Path, err.Error())
		return
	}

	if image.ThumbnailNeeded(srcImage, models.DefaultGthumbWidth) {
		data, err := image.GetThumbnail(srcImage, models.DefaultGthumbWidth)
		if err != nil {
			logger.Errorf("error getting thumbnail for image %s: %s", i.Path, err.Error())
			return
		}

		err = utils.WriteFile(thumbPath, data)
		if err != nil {
			logger.Errorf("error writing thumbnail for image %s: %s", i.Path, err)
		}
	}
}

func (t *ScanTask) calculateChecksum() (string, error) {
	logger.Infof("Calculating checksum for %s...", t.FilePath)
	checksum, err := utils.MD5FromFilePath(t.FilePath)
	if err != nil {
		return "", err
	}
	logger.Debugf("Checksum calculated: %s", checksum)
	return checksum, nil
}

func (t *ScanTask) calculateImageChecksum() (string, error) {
	logger.Infof("Calculating checksum for %s...", image.PathDisplayName(t.FilePath))
	// uses image.CalculateMD5 to read files in zips
	checksum, err := image.CalculateMD5(t.FilePath)
	if err != nil {
		return "", err
	}
	logger.Debugf("Checksum calculated: %s", checksum)
	return checksum, nil
}

func (t *ScanTask) doesPathExist() bool {
	vidExt := config.GetVideoExtensions()
	imgExt := config.GetImageExtensions()
	gExt := config.GetGalleryExtensions()

	if matchExtension(t.FilePath, gExt) {
		qb := models.NewGalleryQueryBuilder()
		gallery, _ := qb.FindByPath(t.FilePath)
		if gallery != nil {
			return true
		}
	} else if matchExtension(t.FilePath, vidExt) {
		qb := models.NewSceneQueryBuilder()
		scene, _ := qb.FindByPath(t.FilePath)
		if scene != nil {
			return true
		}
	} else if matchExtension(t.FilePath, imgExt) {
		qb := models.NewImageQueryBuilder()
		i, _ := qb.FindByPath(t.FilePath)
		if i != nil {
			return true
		}
	}

	return false
}

func walkFilesToScan(s *models.StashConfig, f filepath.WalkFunc) error {
	vidExt := config.GetVideoExtensions()
	imgExt := config.GetImageExtensions()
	gExt := config.GetGalleryExtensions()
	excludeVidRegex := generateRegexps(config.GetExcludes())
	excludeImgRegex := generateRegexps(config.GetImageExcludes())

	return utils.SymWalk(s.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Warnf("error scanning %s: %s", path, err.Error())
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if !s.ExcludeVideo && matchExtension(path, vidExt) && !matchFileRegex(path, excludeVidRegex) {
			return f(path, info, err)
		}

		if !s.ExcludeImage {
			if (matchExtension(path, imgExt) || matchExtension(path, gExt)) && !matchFileRegex(path, excludeImgRegex) {
				return f(path, info, err)
			}
		}

		return nil
	})
}

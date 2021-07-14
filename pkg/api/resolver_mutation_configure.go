package api

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/manager"
	"github.com/stashapp/stash/pkg/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

func (r *mutationResolver) Setup(ctx context.Context, input models.SetupInput) (bool, error) {
	err := manager.GetInstance().Setup(input)
	return err == nil, err
}

func (r *mutationResolver) Migrate(ctx context.Context, input models.MigrateInput) (bool, error) {
	err := manager.GetInstance().Migrate(input)
	return err == nil, err
}

func (r *mutationResolver) ConfigureGeneral(ctx context.Context, input models.ConfigGeneralInput) (*models.ConfigGeneralResult, error) {
	c := config.GetInstance()
	existingPaths := c.GetStashPaths()
	if len(input.Stashes) > 0 {
		for _, s := range input.Stashes {
			// Only validate existence of new paths
			isNew := true
			for _, path := range existingPaths {
				if path.Path == s.Path {
					isNew = false
					break
				}
			}
			if isNew {
				exists, err := utils.DirExists(s.Path)
				if !exists {
					return makeConfigGeneralResult(), err
				}
			}
		}
		c.Set(config.Stash, input.Stashes)
	}

	if input.DatabasePath != nil {
		ext := filepath.Ext(*input.DatabasePath)
		if ext != ".db" && ext != ".sqlite" && ext != ".sqlite3" {
			return makeConfigGeneralResult(), fmt.Errorf("invalid database path, use extension db, sqlite, or sqlite3")
		}
		c.Set(config.Database, input.DatabasePath)
	}

	if input.GeneratedPath != nil {
		if err := utils.EnsureDir(*input.GeneratedPath); err != nil {
			return makeConfigGeneralResult(), err
		}
		c.Set(config.Generated, input.GeneratedPath)
	}

	if input.CachePath != nil {
		if *input.CachePath != "" {
			if err := utils.EnsureDir(*input.CachePath); err != nil {
				return makeConfigGeneralResult(), err
			}
		}
		c.Set(config.Cache, input.CachePath)
	}

	if !input.CalculateMd5 && input.VideoFileNamingAlgorithm == models.HashAlgorithmMd5 {
		return makeConfigGeneralResult(), errors.New("calculateMD5 must be true if using MD5")
	}

	if input.VideoFileNamingAlgorithm != c.GetVideoFileNamingAlgorithm() {
		// validate changing VideoFileNamingAlgorithm
		if err := manager.ValidateVideoFileNamingAlgorithm(r.txnManager, input.VideoFileNamingAlgorithm); err != nil {
			return makeConfigGeneralResult(), err
		}

		c.Set(config.VideoFileNamingAlgorithm, input.VideoFileNamingAlgorithm)
	}

	c.Set(config.CalculateMD5, input.CalculateMd5)

	if input.ParallelTasks != nil {
		c.Set(config.ParallelTasks, *input.ParallelTasks)
	}

	c.Set(config.PreviewAudio, input.PreviewAudio)

	if input.PreviewSegments != nil {
		c.Set(config.PreviewSegments, *input.PreviewSegments)
	}
	if input.PreviewSegmentDuration != nil {
		c.Set(config.PreviewSegmentDuration, *input.PreviewSegmentDuration)
	}
	if input.PreviewExcludeStart != nil {
		c.Set(config.PreviewExcludeStart, *input.PreviewExcludeStart)
	}
	if input.PreviewExcludeEnd != nil {
		c.Set(config.PreviewExcludeEnd, *input.PreviewExcludeEnd)
	}
	if input.PreviewPreset != nil {
		c.Set(config.PreviewPreset, input.PreviewPreset.String())
	}

	if input.MaxTranscodeSize != nil {
		c.Set(config.MaxTranscodeSize, input.MaxTranscodeSize.String())
	}

	if input.MaxStreamingTranscodeSize != nil {
		c.Set(config.MaxStreamingTranscodeSize, input.MaxStreamingTranscodeSize.String())
	}

	if input.Username != nil {
		c.Set(config.Username, input.Username)
	}

	if input.Password != nil {
		// bit of a hack - check if the passed in password is the same as the stored hash
		// and only set if they are different
		currentPWHash := c.GetPasswordHash()

		if *input.Password != currentPWHash {
			c.SetPassword(*input.Password)
		}
	}

	if input.MaxSessionAge != nil {
		c.Set(config.MaxSessionAge, *input.MaxSessionAge)
	}

	if input.LogFile != nil {
		c.Set(config.LogFile, input.LogFile)
	}

	c.Set(config.LogOut, input.LogOut)
	c.Set(config.LogAccess, input.LogAccess)

	if input.LogLevel != c.GetLogLevel() {
		c.Set(config.LogLevel, input.LogLevel)
		logger.SetLogLevel(input.LogLevel)
	}

	if input.Excludes != nil {
		c.Set(config.Exclude, input.Excludes)
	}

	if input.ImageExcludes != nil {
		c.Set(config.ImageExclude, input.ImageExcludes)
	}

	if input.VideoExtensions != nil {
		c.Set(config.VideoExtensions, input.VideoExtensions)
	}

	if input.ImageExtensions != nil {
		c.Set(config.ImageExtensions, input.ImageExtensions)
	}

	if input.GalleryExtensions != nil {
		c.Set(config.GalleryExtensions, input.GalleryExtensions)
	}

	c.Set(config.CreateGalleriesFromFolders, input.CreateGalleriesFromFolders)

	refreshScraperCache := false
	if input.ScraperUserAgent != nil {
		c.Set(config.ScraperUserAgent, input.ScraperUserAgent)
		refreshScraperCache = true
	}

	if input.ScraperCDPPath != nil {
		c.Set(config.ScraperCDPPath, input.ScraperCDPPath)
		refreshScraperCache = true
	}

	c.Set(config.ScraperCertCheck, input.ScraperCertCheck)

	if input.StashBoxes != nil {
		if err := c.ValidateStashBoxes(input.StashBoxes); err != nil {
			return nil, err
		}
		c.Set(config.StashBoxes, input.StashBoxes)
	}

	if err := c.Write(); err != nil {
		return makeConfigGeneralResult(), err
	}

	manager.GetInstance().RefreshConfig()
	if refreshScraperCache {
		manager.GetInstance().RefreshScraperCache()
	}

	return makeConfigGeneralResult(), nil
}

func (r *mutationResolver) ConfigureInterface(ctx context.Context, input models.ConfigInterfaceInput) (*models.ConfigInterfaceResult, error) {
	c := config.GetInstance()
	if input.MenuItems != nil {
		c.Set(config.MenuItems, input.MenuItems)
	}

	if input.SoundOnPreview != nil {
		c.Set(config.SoundOnPreview, *input.SoundOnPreview)
	}

	if input.WallShowTitle != nil {
		c.Set(config.WallShowTitle, *input.WallShowTitle)
	}

	if input.CustomPerformerImageLocation != nil {
		c.Set(config.CustomPerformerImageLocation, *input.CustomPerformerImageLocation)
	}

	if input.WallPlayback != nil {
		c.Set(config.WallPlayback, *input.WallPlayback)
	}

	if input.MaximumLoopDuration != nil {
		c.Set(config.MaximumLoopDuration, *input.MaximumLoopDuration)
	}

	if input.AutostartVideo != nil {
		c.Set(config.AutostartVideo, *input.AutostartVideo)
	}

	if input.ShowStudioAsText != nil {
		c.Set(config.ShowStudioAsText, *input.ShowStudioAsText)
	}

	if input.Language != nil {
		c.Set(config.Language, *input.Language)
	}

	if input.SlideshowDelay != nil {
		c.Set(config.SlideshowDelay, *input.SlideshowDelay)
	}

	css := ""

	if input.CSS != nil {
		css = *input.CSS
	}

	c.SetCSS(css)

	if input.CSSEnabled != nil {
		c.Set(config.CSSEnabled, *input.CSSEnabled)
	}

	if input.HandyKey != nil {
		c.Set(config.HandyKey, *input.HandyKey)
	}

	if err := c.Write(); err != nil {
		return makeConfigInterfaceResult(), err
	}

	return makeConfigInterfaceResult(), nil
}

func (r *mutationResolver) ConfigureDlna(ctx context.Context, input models.ConfigDLNAInput) (*models.ConfigDLNAResult, error) {
	c := config.GetInstance()

	if input.ServerName != nil {
		c.Set(config.DLNAServerName, *input.ServerName)
	}

	c.Set(config.DLNADefaultIPWhitelist, input.WhitelistedIPs)

	currentDLNAEnabled := c.GetDLNADefaultEnabled()
	if input.Enabled != nil && *input.Enabled != currentDLNAEnabled {
		c.Set(config.DLNADefaultEnabled, *input.Enabled)

		// start/stop the DLNA service as needed
		dlnaService := manager.GetInstance().DLNAService
		if !*input.Enabled && dlnaService.IsRunning() {
			dlnaService.Stop(nil)
		} else if *input.Enabled && !dlnaService.IsRunning() {
			dlnaService.Start(nil)
		}
	}

	c.Set(config.DLNAInterfaces, input.Interfaces)

	if err := c.Write(); err != nil {
		return makeConfigDLNAResult(), err
	}

	return makeConfigDLNAResult(), nil
}

func (r *mutationResolver) GenerateAPIKey(ctx context.Context, input models.GenerateAPIKeyInput) (string, error) {
	c := config.GetInstance()

	var newAPIKey string
	if input.Clear == nil || !*input.Clear {
		username := c.GetUsername()
		if username != "" {
			var err error
			newAPIKey, err = manager.GenerateAPIKey(username)
			if err != nil {
				return "", err
			}
		}
	}

	c.Set(config.ApiKey, newAPIKey)
	if err := c.Write(); err != nil {
		return newAPIKey, err
	}

	return newAPIKey, nil
}

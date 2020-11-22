package manager

import (
	"net"
	"sync"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/manager/config"
	"github.com/stashapp/stash/pkg/manager/paths"
	"github.com/stashapp/stash/pkg/plugin"
	"github.com/stashapp/stash/pkg/scraper"
	"github.com/stashapp/stash/pkg/utils"
)

type singleton struct {
	Status TaskStatus
	Paths  *paths.Paths

	FFMPEGPath  string
	FFProbePath string

	PluginCache  *plugin.Cache
	ScraperCache *scraper.Cache

	DownloadStore *DownloadStore
}

var instance *singleton
var once sync.Once

type flagStruct struct {
	configFilePath string
}

var flags = flagStruct{}

func GetInstance() *singleton {
	Initialize()
	return instance
}

func Initialize() *singleton {
	once.Do(func() {
		_ = utils.EnsureDir(paths.GetConfigDirectory())
		initFlags()
		initConfig()
		initLog()
		initEnvs()
		instance = &singleton{
			Status: TaskStatus{Status: Idle, Progress: -1},
			Paths:  paths.NewPaths(),

			PluginCache:  initPluginCache(),
			ScraperCache: initScraperCache(),

			DownloadStore: NewDownloadStore(),
		}

		instance.RefreshConfig()

		// clear the downloads and tmp directories
		utils.EmptyDir(instance.Paths.Generated.Downloads)
		utils.EmptyDir(instance.Paths.Generated.Tmp)

		initFFMPEG()
	})

	return instance
}

func initConfig() {
	// The config file is called config.  Leave off the file extension.
	viper.SetConfigName("config")

	if flagConfigFileExists, _ := utils.FileExists(flags.configFilePath); flagConfigFileExists {
		viper.SetConfigFile(flags.configFilePath)
	}
	viper.AddConfigPath(".")            // Look for config in the working directory
	viper.AddConfigPath("$HOME/.stash") // Look for the config in the home directory

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		_ = utils.Touch(paths.GetDefaultConfigFilePath())
		if err = viper.ReadInConfig(); err != nil {
			panic(err)
		}
	}
	logger.Infof("using config file: %s", viper.ConfigFileUsed())

	config.SetInitialConfig()

	viper.SetDefault(config.Database, paths.GetDefaultDatabaseFilePath())

	// Set generated to the metadata path for backwards compat
	viper.SetDefault(config.Generated, viper.GetString(config.Metadata))

	// Set default scrapers and plugins paths
	viper.SetDefault(config.ScrapersPath, config.GetDefaultScrapersPath())
	viper.SetDefault(config.PluginsPath, config.GetDefaultPluginsPath())

	// Disabling config watching due to race condition issue
	// See: https://github.com/spf13/viper/issues/174
	// Changes to the config outside the system will require a restart
	// Watch for changes
	// viper.WatchConfig()
	// viper.OnConfigChange(func(e fsnotify.Event) {
	// 	fmt.Println("Config file changed:", e.Name)
	// 	instance.refreshConfig()
	// })

	//viper.Set("stash", []string{"/", "/stuff"})
	//viper.WriteConfig()
}

func initFlags() {
	pflag.IP("host", net.IPv4(0, 0, 0, 0), "ip address for the host")
	pflag.Int("port", 9999, "port to serve from")
	pflag.StringVarP(&flags.configFilePath, "config", "c", "", "config file to use")

	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		logger.Infof("failed to bind flags: %s", err.Error())
	}
}

func initEnvs() {
	viper.SetEnvPrefix("stash")    // will be uppercased automatically
	viper.BindEnv("host")          // STASH_HOST
	viper.BindEnv("port")          // STASH_PORT
	viper.BindEnv("external_host") // STASH_EXTERNAL_HOST
	viper.BindEnv("generated")     // STASH_GENERATED
	viper.BindEnv("metadata")      // STASH_METADATA
	viper.BindEnv("cache")         // STASH_CACHE

	// only set stash config flag if not already set
	if config.GetStashPaths() == nil {
		viper.BindEnv("stash") // STASH_STASH
	}
}

func initFFMPEG() {
	configDirectory := paths.GetConfigDirectory()
	ffmpegPath, ffprobePath := ffmpeg.GetPaths(configDirectory)
	if ffmpegPath == "" || ffprobePath == "" {
		logger.Infof("couldn't find FFMPEG, attempting to download it")
		if err := ffmpeg.Download(configDirectory); err != nil {
			msg := `Unable to locate / automatically download FFMPEG

Check the readme for download links.
The FFMPEG and FFProbe binaries should be placed in %s

The error was: %s
`
			logger.Fatalf(msg, configDirectory, err)
		} else {
			// After download get new paths for ffmpeg and ffprobe
			ffmpegPath, ffprobePath = ffmpeg.GetPaths(configDirectory)
		}
	}

	instance.FFMPEGPath = ffmpegPath
	instance.FFProbePath = ffprobePath
}

func initLog() {
	logger.Init(config.GetLogFile(), config.GetLogOut(), config.GetLogLevel())
}

func initPluginCache() *plugin.Cache {
	ret, err := plugin.NewCache(config.GetPluginsPath())

	if err != nil {
		logger.Errorf("Error reading plugin configs: %s", err.Error())
	}

	return ret
}

// initScraperCache initializes a new scraper cache and returns it.
func initScraperCache() *scraper.Cache {
	scraperConfig := scraper.GlobalConfig{
		Path:      config.GetScrapersPath(),
		UserAgent: config.GetScraperUserAgent(),
		CDPPath:   config.GetScraperCDPPath(),
	}
	ret, err := scraper.NewCache(scraperConfig)

	if err != nil {
		logger.Errorf("Error reading scraper configs: %s", err.Error())
	}

	return ret
}

func (s *singleton) RefreshConfig() {
	s.Paths = paths.NewPaths()
	if config.IsValid() {
		utils.EnsureDir(s.Paths.Generated.Screenshots)
		utils.EnsureDir(s.Paths.Generated.Vtt)
		utils.EnsureDir(s.Paths.Generated.Markers)
		utils.EnsureDir(s.Paths.Generated.Transcodes)
		utils.EnsureDir(s.Paths.Generated.Downloads)
		paths.EnsureJSONDirs(config.GetMetadataPath())
	}
}

// RefreshScraperCache refreshes the scraper cache. Call this when scraper
// configuration changes.
func (s *singleton) RefreshScraperCache() {
	s.ScraperCache = initScraperCache()
}

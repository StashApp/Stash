package database

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fvbommel/sortorder"
	"github.com/gobuffalo/packr/v2"
	"github.com/golang-migrate/migrate/v4"
	sqlite3mig "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/utils"
)

var DB *sqlx.DB
var WriteMu *sync.Mutex
var dbPath string
var appSchemaVersion uint = 26
var databaseSchemaVersion uint

var (
	// ErrMigrationNeeded indicates that a database migration is needed
	// before the database can be initialized
	ErrMigrationNeeded = errors.New("database migration required")

	// ErrDatabaseNotInitialized indicates that the database is not
	// initialized, usually due to an incomplete configuration.
	ErrDatabaseNotInitialized = errors.New("database not initialized")
)

const sqlite3Driver = "sqlite3ex"

// Ready returns an error if the database is not ready to begin transactions.
func Ready() error {
	if DB == nil {
		return ErrDatabaseNotInitialized
	}

	return nil
}

func init() {
	// register custom driver with regexp function
	registerCustomDriver()
}

// Initialize initializes the database. If the database is new, then it
// performs a full migration to the latest schema version. Otherwise, any
// necessary migrations must be run separately using RunMigrations.
// Returns true if the database is new.
func Initialize(databasePath string) error {
	dbPath = databasePath

	if err := getDatabaseSchemaVersion(); err != nil {
		return fmt.Errorf("error getting database schema version: %s", err.Error())
	}

	if databaseSchemaVersion == 0 {
		// new database, just run the migrations
		if err := RunMigrations(); err != nil {
			return fmt.Errorf("error running initial schema migrations: %s", err.Error())
		}
		// RunMigrations calls Initialise. Just return
		return nil
	} else {
		if databaseSchemaVersion > appSchemaVersion {
			panic(fmt.Sprintf("Database schema version %d is incompatible with required schema version %d", databaseSchemaVersion, appSchemaVersion))
		}

		// if migration is needed, then don't open the connection
		if NeedsMigration() {
			logger.Warnf("Database schema version %d does not match required schema version %d.", databaseSchemaVersion, appSchemaVersion)
			return nil
		}
	}

	const disableForeignKeys = false
	DB = open(databasePath, disableForeignKeys)
	WriteMu = &sync.Mutex{}

	return nil
}

func open(databasePath string, disableForeignKeys bool) *sqlx.DB {
	// https://github.com/mattn/go-sqlite3
	url := "file:" + databasePath + "?_journal=WAL"
	if !disableForeignKeys {
		url += "&_fk=true"
	}

	conn, err := sqlx.Open(sqlite3Driver, url)
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(4)
	conn.SetConnMaxLifetime(30 * time.Second)
	if err != nil {
		logger.Fatalf("db.Open(): %q\n", err)
	}

	return conn
}

func Reset(databasePath string) error {
	err := DB.Close()

	if err != nil {
		return errors.New("Error closing database: " + err.Error())
	}

	err = os.Remove(databasePath)
	if err != nil {
		return errors.New("Error removing database: " + err.Error())
	}

	// remove the -shm, -wal files ( if they exist )
	walFiles := []string{databasePath + "-shm", databasePath + "-wal"}
	for _, wf := range walFiles {
		if exists, _ := utils.FileExists(wf); exists {
			err = os.Remove(wf)
			if err != nil {
				return errors.New("Error removing database: " + err.Error())
			}
		}
	}

	Initialize(databasePath)
	return nil
}

// Backup the database. If db is nil, then uses the existing database
// connection.
func Backup(db *sqlx.DB, backupPath string) error {
	if db == nil {
		var err error
		db, err = sqlx.Connect(sqlite3Driver, "file:"+dbPath+"?_fk=true")
		if err != nil {
			return fmt.Errorf("Open database %s failed:%s", dbPath, err)
		}
		defer db.Close()
	}

	logger.Infof("Backing up database into: %s", backupPath)
	_, err := db.Exec(`VACUUM INTO "` + backupPath + `"`)
	if err != nil {
		return fmt.Errorf("Vacuum failed: %s", err)
	}

	return nil
}

func RestoreFromBackup(backupPath string) error {
	logger.Infof("Restoring backup database %s into %s", backupPath, dbPath)
	return os.Rename(backupPath, dbPath)
}

// Migrate the database
func NeedsMigration() bool {
	return databaseSchemaVersion != appSchemaVersion
}

func AppSchemaVersion() uint {
	return appSchemaVersion
}

func DatabasePath() string {
	return dbPath
}

func DatabaseBackupPath() string {
	return fmt.Sprintf("%s.%d.%s", dbPath, databaseSchemaVersion, time.Now().Format("20060102_150405"))
}

func Version() uint {
	return databaseSchemaVersion
}

func getMigrate() (*migrate.Migrate, error) {
	migrationsBox := packr.New("Migrations Box", "./migrations")
	packrSource := &Packr2Source{
		Box:        migrationsBox,
		Migrations: source.NewMigrations(),
	}

	databasePath := utils.FixWindowsPath(dbPath)
	s, _ := WithInstance(packrSource)

	const disableForeignKeys = true
	conn := open(databasePath, disableForeignKeys)

	driver, err := sqlite3mig.WithInstance(conn.DB, &sqlite3mig.Config{})
	if err != nil {
		return nil, err
	}

	// use sqlite3Driver so that migration has access to durationToTinyInt
	return migrate.NewWithInstance(
		"packr2",
		s,
		databasePath,
		driver,
	)
}

func getDatabaseSchemaVersion() error {
	m, err := getMigrate()
	if err != nil {
		return err
	}

	databaseSchemaVersion, _, _ = m.Version()
	m.Close()
	return nil
}

// Migrate the database
func RunMigrations() error {
	m, err := getMigrate()
	if err != nil {
		panic(err.Error())
	}

	databaseSchemaVersion, _, _ = m.Version()
	stepNumber := appSchemaVersion - databaseSchemaVersion
	if stepNumber != 0 {
		logger.Infof("Migrating database from version %d to %d", databaseSchemaVersion, appSchemaVersion)
		err = m.Steps(int(stepNumber))
		if err != nil {
			// migration failed
			logger.Errorf("Error migrating database: %s", err.Error())
			m.Close()
			return err
		}
	}

	m.Close()

	// re-initialise the database
	Initialize(dbPath)

	// run a vacuum on the database
	logger.Info("Performing vacuum on database")
	_, err = DB.Exec("VACUUM")
	if err != nil {
		logger.Warnf("error while performing post-migration vacuum: %s", err.Error())
	}

	return nil
}

func registerCustomDriver() {
	sql.Register(sqlite3Driver,
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				funcs := map[string]interface{}{
					"regexp":            regexFn,
					"durationToTinyInt": durationToTinyIntFn,
				}

				for name, fn := range funcs {
					if err := conn.RegisterFunc(name, fn, true); err != nil {
						return fmt.Errorf("Error registering function %s: %s", name, err.Error())
					}
				}

				// COLLATE NATURAL_CS - Case sensitive natural sort
				err := conn.RegisterCollation("NATURAL_CS", func(s string, s2 string) int {
					if sortorder.NaturalLess(s, s2) {
						return -1
					} else {
						return 1
					}
				})

				if err != nil {
					return fmt.Errorf("Error registering natural sort collation: %s", err.Error())
				}

				return nil
			},
		},
	)
}

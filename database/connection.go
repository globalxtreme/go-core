package xtremedb

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	"github.com/natefinch/lumberjack"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const POSTGRESQL_DRIVER = "pgsql"
const MYSQL_DRIVER = "mysql"

const POSTGRESQL_COLLATE = "en_US.utf8"
const MYSQL_COLLATE = "utf8mb4_unicode_ci"

type DBConf struct {
	Driver          string
	Host            string
	Port            any
	Owner           string
	Username        string
	Password        string
	Database        string
	Charset         string
	ParseTime       bool
	Loc             string
	Collation       string
	TimeZone        string
	MaxOpenCons     int
	MaxIdleCons     int
	MaxLifetimeCons time.Duration
}

func GetDBDrivers() []string {
	return []string{
		POSTGRESQL_DRIVER,
		MYSQL_DRIVER,
	}
}

func Connect(conn DBConf) (*gorm.DB, func()) {
	var driver *gorm.DB

	switch conn.Driver {
	case POSTGRESQL_DRIVER:
		driver = postgresqlConnection(conn)
		break
	default:
		driver = mysqlConnection(conn)
		break
	}

	sqlDB, err := driver.DB()
	if err != nil {
		panic(err)
	}

	if conn.MaxOpenCons == 0 {
		conn.MaxOpenCons = 1000
	}

	if conn.MaxIdleCons == 0 {
		conn.MaxIdleCons = 50
	}

	if conn.MaxLifetimeCons == 0 {
		conn.MaxLifetimeCons = 10 * time.Minute
	}

	sqlDB.SetMaxOpenConns(conn.MaxOpenCons)
	sqlDB.SetMaxIdleConns(conn.MaxIdleCons)
	sqlDB.SetConnMaxLifetime(conn.MaxLifetimeCons)

	DBClose := func() {
		sqlDB.Close()
	}

	return driver, DBClose
}

func SetMigration(conn *gorm.DB, collate string) *gorm.DB {
	return conn.Set("gorm:table_options", fmt.Sprintf("COLLATE=%s", collate))
}

func WithMultipleTransactions(dbs map[string]*gorm.DB, fc func(txs map[string]*gorm.DB) error) (err error) {
	panicked := true
	txs := make(map[string]*gorm.DB)

	for name, db := range dbs {
		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}
		txs[name] = tx
	}

	defer func() {
		if r := recover(); r != nil {
			for _, tx := range txs {
				_ = tx.Rollback()
			}
			panic(r)
		}
		if panicked || err != nil {
			for _, tx := range txs {
				_ = tx.Rollback()
			}
		} else {
			for _, tx := range txs {
				if commitErr := tx.Commit().Error; commitErr != nil {
					for _, tx := range txs {
						_ = tx.Rollback()
					}
					err = commitErr
					return
				}
			}
		}
	}()

	err = fc(txs)
	panicked = false
	return
}

func postgresqlConnection(conn DBConf) *gorm.DB {
	if len(conn.TimeZone) == 0 {
		conn.TimeZone = "Asia/Kuala_Lumpur"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=%s",
		conn.Host, conn.Username, conn.Password, conn.Database, conn.Port, conn.TimeZone)
	driver, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: setNewLogger(conn.Driver)})
	if err != nil {
		panic(err)
	}

	return driver
}

func mysqlConnection(conn DBConf) *gorm.DB {
	option := "?"

	if len(conn.Charset) > 0 {
		option += "charset=" + conn.Charset
	} else {
		option += "charset=utf8mb4"
	}

	if conn.ParseTime {
		option += "&parseTime=True"
	} else {
		option += "&parseTime=False"
	}

	if len(conn.Loc) > 0 {
		option += "&loc=" + conn.Loc
	} else {
		option += "&loc=Local"
	}

	if len(conn.Collation) > 0 {
		option += "&collation=" + conn.Collation
	} else {
		option += "&collation=utf8mb4_unicode_ci"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s%s",
		conn.Username, conn.Password, conn.Host, conn.Port, conn.Database, option)
	driver, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: setNewLogger(conn.Driver)})
	if err != nil {
		panic(err)
	}

	return driver
}

func setNewLogger(driver string) logger.Interface {
	storageDir := os.Getenv("STORAGE_DIR") + "/logs"
	xtremepkg.CheckAndCreateDirectory(storageDir)

	logDays := os.Getenv("LOG_DAYS")
	if logDays == "" {
		logDays = "7"
	}

	maxAge, err := strconv.Atoi(logDays)
	if err != nil {
		maxAge = 7
	}

	logFile := &lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/%s.log", storageDir, driver),
		MaxSize:    100, // megabytes
		MaxBackups: 30,
		MaxAge:     maxAge, // days
		Compress:   true,
	}

	newLogger := logger.New(
		log.New(logFile, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	return newLogger
}

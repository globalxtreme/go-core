package database

import (
	"fmt"
	xtremepkg "github.com/globalxtreme/go-core/pkg"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

const POSTGRESQL_DRIVER = "pgsql"
const MYSQL_DRIVER = "mysql"

const POSTGRESQL_COLLATE = "en_US.utf8"
const MYSQL_COLLATE = "utf8mb4_unicode_ci"

type DBConf struct {
	Driver    string
	Host      string
	Port      any
	Owner     string
	Username  string
	Password  string
	Database  string
	Charset   string
	ParseTime bool
	Loc       string
	Collation string
	TimeZone  string
}

func Connect(conn DBConf) *gorm.DB {
	var driver *gorm.DB

	switch conn.Driver {
	case POSTGRESQL_DRIVER:
		driver = postgresqlConnection(conn)
		break
	default:
		driver = mysqlConnection(conn)
		break
	}

	return driver
}

func SetMigration(conn *gorm.DB, collate string) *gorm.DB {
	return conn.Set("gorm:table_options", fmt.Sprintf("COLLATE=%s", collate))
}

func postgresqlConnection(conn DBConf) *gorm.DB {
	if len(conn.TimeZone) == 0 {
		conn.TimeZone = "Asia/Kuala_Lumpur"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=%s",
		conn.Host, conn.Username, conn.Password, conn.Database, conn.Port, conn.TimeZone)
	driver, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: setNewLogger()})
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
	driver, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: setNewLogger()})
	if err != nil {
		panic(err)
	}

	return driver
}

func setNewLogger() logger.Interface {
	storageDir := os.Getenv("STORAGE_DIR") + "/logs"
	xtremepkg.CheckAndCreateDirectory(storageDir)

	filename := time.Now().Format("2006-01-02") + ".log"
	logFile, err := os.OpenFile(storageDir+"/"+filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
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

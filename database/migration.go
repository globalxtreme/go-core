package xtremedb

import (
	"fmt"
	xtrememodel "github.com/globalxtreme/go-core/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"log"
	"os"
	"time"
)

type Migration interface {
	Reference() string
	Tables() []Table
	Columns() []Column
}

type Table struct {
	Connection  *gorm.DB
	CreateTable schema.Tabler
	RenameTable Rename
	DropTable   string
	Collate     string
	Owner       string
}

type Column struct {
	Connection    *gorm.DB
	Model         schema.Tabler
	RenameColumns []Rename
	AddColumns    []string
	DropColumns   []string
	AlterColumns  []string
	Collate       string
}

type Rename struct {
	Old string
	New string
}

func Migrate(conn *gorm.DB, migrations []Migration) {
	var err error
	var migration *gorm.DB
	var migrator gorm.Migrator

	checkAndSetMigrationTable(conn)

	for _, mgr := range migrations {
		var countReference int64
		conn.Model(&xtrememodel.Migration{}).Where("reference = ?", mgr.Reference()).Count(&countReference)
		if countReference > 0 {
			continue
		}

		for _, table := range mgr.Tables() {
			if len(table.Collate) > 0 {
				migration = SetMigration(table.Connection, table.Collate)
			} else {
				migration = table.Connection
			}

			if table.CreateTable != nil {
				migrator = migration.Table(table.CreateTable.TableName()).Migrator()
				if !migrator.HasTable(table.CreateTable) {
					err = migrator.CreateTable(table.CreateTable)
					if err != nil {
						log.Panicf("CREATE CREATE: %v", err)
					}

					if len(table.Owner) > 0 {
						err = table.Connection.Exec(fmt.Sprintf("ALTER TABLE %s OWNER TO %s", table.CreateTable.TableName(), table.Owner)).Error
						if err != nil {
							log.Panicf("CHANGE OWNER: %v", err)
						}
					}
				}
			}

			if len(table.RenameTable.Old) > 0 {
				migrator = migration.Table(table.RenameTable.Old).Migrator()
				if migrator.HasTable(table.RenameTable.Old) {
					err = migrator.RenameTable(table.RenameTable.Old, table.RenameTable.New)
					if err != nil {
						log.Panicf("RENAME TABLE: %v", err)
					}
				}
			}

			if len(table.DropTable) > 0 {
				migrator = migration.Table(table.DropTable).Migrator()
				if migrator.HasTable(table.DropTable) {
					err = migrator.DropTable(table.DropTable)
					if err != nil {
						log.Panicf("DROP TABLE: %v", err)
					}
				}
			}
		}

		for _, column := range mgr.Columns() {
			if len(column.Collate) > 0 {
				migration = SetMigration(column.Connection, column.Collate)
			} else {
				migration = column.Connection
			}

			migrator = migration.Table(column.Model.TableName()).Migrator()

			if len(column.RenameColumns) > 0 {
				for _, rename := range column.RenameColumns {
					if migrator.HasColumn(column.Model, rename.Old) {
						err = migrator.RenameColumn(column.Model, rename.Old, rename.New)
						if err != nil {
							log.Panicf("RENAME COLUMN: %v", err)
						}
					}
				}
			}

			if len(column.AddColumns) > 0 {
				for _, add := range column.AddColumns {
					if !migrator.HasColumn(column.Model, add) {
						err = migrator.AddColumn(column.Model, add)
						if err != nil {
							log.Panicf("ADD COLUMN: %v", err)
						}
					}
				}
			}

			if len(column.DropColumns) > 0 {
				for _, drop := range column.DropColumns {
					if migrator.HasColumn(column.Model, drop) {
						err = migrator.DropColumn(column.Model, drop)
						if err != nil {
							log.Panicf("DROP COLUMN: %v", err)
						}
					}
				}
			}

			if len(column.AlterColumns) > 0 {
				for _, alter := range column.AlterColumns {
					if migrator.HasColumn(column.Model, alter) {
						err = migrator.AlterColumn(column.Model, alter)
						if err != nil {
							log.Panicf("ALTER COLUMN: %v", err)
						}
					}
				}
			}
		}

		err = conn.Create(&xtrememodel.Migration{Reference: mgr.Reference()}).Error
		if err != nil {
			log.Panicf("Could not save reference to migrations. %v", err)
		}

		fmt.Printf("%-23s %s\n", time.Now().Format("2006-01-02 15:04:05"), mgr.Reference())
	}
}

func checkAndSetMigrationTable(conn *gorm.DB) {
	owner := os.Getenv("DB_OWNER")
	model := xtrememodel.Migration{}

	migrator := conn.Table(model.TableName()).Migrator()
	if !migrator.HasTable(model) {
		err := migrator.CreateTable(model)
		if err != nil {
			log.Panicf("CREATE CREATE: %v", err)
		}

		if len(owner) > 0 {
			err = conn.Exec(fmt.Sprintf("ALTER TABLE %s OWNER TO %s", model.TableName(), owner)).Error
			if err != nil {
				log.Panicf("CHANGE OWNER: %v", err)
			}
		}
	}
}

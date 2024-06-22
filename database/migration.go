package database

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"log"
)

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

func Migrate(tables []Table, columns []Column) {
	var err error
	var migration *gorm.DB
	var migrator gorm.Migrator

	for _, table := range tables {
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

	for _, column := range columns {
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
}

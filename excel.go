package xtremecore

import (
	"github.com/xuri/excelize/v2"
	"log"
	"strings"
)

type Excel struct {
	File       *excelize.File
	Sheets     []string
	Properties [][][]interface{}
	IsPublic   bool
	prepared   bool
}

type ColWidth struct {
	Cells string
	Width float64
}

type RowHeight struct {
	Row    int
	Height float64
}

func (ex *Excel) NewFile() *Excel {
	totalProperties := len(ex.Properties)
	if totalProperties > 1 {
		if len(ex.Sheets) != totalProperties {
			log.Panicf("Your sheets and properties not match!!")
		}
	}

	ex.File = excelize.NewFile()

	sKey := 0
	for key, sheet := range ex.Sheets {
		rename := false

		if sKey == 0 {
			if sheet != "Sheet1" {
				rename = true
			}
		}

		if rename {
			ex.File.SetSheetName("Sheet1", sheet)
		} else {
			ex.File.NewSheet(sheet)
		}

		rKey := 1
		for _, property := range ex.Properties[key] {
			excelAddRow(sheet, ex.File, rKey, property)
			rKey++
		}

		sKey++
	}

	ex.prepared = true

	return ex
}

func (ex *Excel) MergeCells(cells ...string) *Excel {
	if len(cells) == 0 {
		log.Panicf("[MERGE-CELL] Your ceels is null!!")
	}

	for _, cell := range cells {
		firstCell, secondCell := excelSplitCells(cell)
		for _, sheet := range ex.Sheets {
			ex.File.MergeCell(sheet, firstCell, secondCell)
		}
	}

	return ex
}

func (ex *Excel) SetWidthCols(cWidths []ColWidth) *Excel {
	if len(cWidths) == 0 {
		log.Panicf("[COL-WIDTH] Your col options is null!!")
	}

	for _, cWidth := range cWidths {
		firstCell, secondCell := excelSplitCells(cWidth.Cells)
		for _, sheet := range ex.Sheets {
			ex.File.SetColWidth(sheet, firstCell, secondCell, cWidth.Width)
		}
	}

	return ex
}

func (ex *Excel) SetHeightRows(rHeights []RowHeight) *Excel {
	if len(rHeights) == 0 {
		log.Panicf("[HEIGHT-ROW] Your row options is null!!")
	}

	for _, rHeight := range rHeights {
		for _, sheet := range ex.Sheets {
			ex.File.SetRowHeight(sheet, rHeight.Row, rHeight.Height)
		}
	}

	return ex
}

func (ex *Excel) SetStyle(dataStyle *excelize.Style, cells ...string) *Excel {
	if len(cells) == 0 {
		log.Panicf("[STYLE] Your ceels is null!!")
	}

	styleID, err := ex.File.NewStyle(dataStyle)
	if err != nil {
		log.Panicf("Unable to create new style: %s", err)
	}

	for _, cell := range cells {
		firstCell, secondCell := excelSplitCells(cell)
		for _, sheet := range ex.Sheets {
			ex.File.SetCellStyle(sheet, firstCell, secondCell, styleID)
		}
	}

	return ex
}

func (ex *Excel) Save(path string, filename string) error {
	if !ex.prepared {
		log.Panicf("Your sheets not yet prepared!! Please call .NewFile()")
	}

	var storagePath string
	if ex.IsPublic {
		storagePath = SetStorageAppPublicDir(path)
	} else {
		storagePath = SetStorageAppDir(path)
	}

	CheckAndCreateDirectory(storagePath)

	err := ex.File.SaveAs(storagePath + filename)
	if err != nil {
		return err
	}

	return nil
}

func excelSplitCells(cells string) (string, string) {
	splitCell := strings.Split(cells, ":")
	firstCell := splitCell[0]
	secondCell := firstCell

	if len(splitCell) > 1 {
		secondCell = splitCell[1]
	}

	return firstCell, secondCell
}

func excelAddRow(sheet string, f *excelize.File, rowIndex int, data []interface{}) {
	columnIndex := 1
	for _, value := range data {
		cell, _ := excelize.CoordinatesToCellName(columnIndex, rowIndex)
		f.SetCellValue(sheet, cell, value)
		columnIndex++
	}
}

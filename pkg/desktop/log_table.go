package desktop

import (
	"github.com/andlabs/ui"
	"github.com/emirpasic/gods/lists/arraylist"
)

type LogTable struct {
	numRows int
	list    arraylist.List
}

func (lt *LogTable) ColumnTypes(m *ui.TableModel) []ui.TableValue {
	return []ui.TableValue{
		ui.TableString(""),
		ui.TableString(""),
	}
}

func (lt *LogTable) NumRows(m *ui.TableModel) int {
	return lt.numRows
}

func (lt *LogTable) CellValue(m *ui.TableModel, row, column int) ui.TableValue {
	return ui.TableString("TODO") //TODO
}

func (lt *LogTable) SetCellValue(m *ui.TableModel, row, column int, value ui.TableValue) {

}

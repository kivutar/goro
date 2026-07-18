package ui

import (
	"image"

	"github.com/gogpu/ui/state"
	"github.com/kivutar/goro/ui/rotheme"
)

type itemTableRow struct {
	name string
	icon image.Image
}

func itemTableView(
	rows []itemTableRow,
	title string,
	rowHeight float32,
	headerHeight float32,
	emptyText string,
	scroll state.Signal[float32],
	selectedRow int,
	onRowClick func(int),
) *rotheme.TableViewWidget {
	options := []rotheme.TableViewOption{
		rotheme.TableViewColumns([]rotheme.TableViewColumn{
			{Key: "item", Title: title, Flex: 1, MinWidth: 120},
		}),
		rotheme.TableViewRowCount(len(rows)),
		rotheme.TableViewRowHeight(rowHeight),
		rotheme.TableViewHeaderHeight(headerHeight),
		rotheme.TableViewEmptyText(emptyText),
		rotheme.TableViewScrollYSignal(scroll),
		rotheme.TableViewInvalidateHover(false),
		rotheme.TableViewDispatchHoverToCells(false),
		rotheme.TableViewBuildSimpleCell(func(cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
			if cell.Row < 0 || cell.Row >= len(rows) {
				return rotheme.TableViewSimpleCell{Hidden: true}
			}
			return rotheme.TableViewSimpleCell{
				Icon: rows[cell.Row].icon,
				Text: rows[cell.Row].name,
			}
		}),
	}
	if onRowClick != nil {
		options = append(options,
			rotheme.TableViewSelectedRow(state.NewSignal[int](selectedRow)),
			rotheme.TableViewOnRowClick(func(row int) {
				if row >= 0 && row < len(rows) {
					onRowClick(row)
				}
			}),
		)
	}
	return rotheme.TableView(options...)
}

package headersearch_test

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"tooling/headersearch"
)

func ExampleExtractHeaders() {
	workbook := excelize.NewFile()
	sheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	_ = workbook.SetSheetName(sheet, "Report")
	_ = workbook.SetCellStr("Report", "A1", "Fund Information")
	_ = workbook.SetCellStr("Report", "A2", "Fund Name")
	_ = workbook.SetCellStr("Report", "B2", "Fund Inception Date")
	_ = workbook.MergeCell("Report", "A1", "B1")

	table, err := headersearch.ExtractHeaders(workbook, headersearch.ExtractOptions{
		Sheet:           "Report",
		Anchor:          "Fund Inception Date",
		ParentDirection: headersearch.DirectionUp,
		MaxHeaderDepth:  3,
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(table.AnchorPosition.Axis)
	fmt.Println(table.Headers[1].Path)
	// Output:
	// B2
	// [Fund Information Fund Inception Date]
}

package files

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"github.com/xuri/excelize/v2"
)

// CSVExcelConnector reads CSV and Excel (xlsx/xls) files.
// Config keys: path (file path)
type CSVExcelConnector struct {
	path string
}

func (c *CSVExcelConnector) SourceType() string { return "csv_excel" }

func (c *CSVExcelConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.path = fmt.Sprintf("%v", cfg["path"])
	if c.path == "" || c.path == "<nil>" {
		return fmt.Errorf("csv_excel: path is required")
	}
	return nil
}

func (c *CSVExcelConnector) Close() error { return nil }

func (c *CSVExcelConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		lower := strings.ToLower(c.path)
		if strings.HasSuffix(lower, ".xlsx") || strings.HasSuffix(lower, ".xls") {
			if err := c.streamExcel(ctx, out); err != nil {
				errc <- err
			}
		} else {
			if err := c.streamCSV(ctx, out); err != nil {
				errc <- err
			}
		}
	}()
	return out, errc
}

func (c *CSVExcelConnector) streamCSV(ctx context.Context, out chan<- connectors.FieldRecord) error {
	f, err := os.Open(c.path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(io.LimitReader(f, 100*1024*1024))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	headers, err := r.Read()
	if err != nil {
		return err
	}

	rowNum := 0
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		rowNum++
		for i, val := range record {
			if val == "" || i >= len(headers) {
				continue
			}
			select {
			case out <- connectors.FieldRecord{
				Value:        val,
				FieldName:    headers[i],
				SourcePath:   fmt.Sprintf("%s:row_%d.%s", c.path, rowNum, headers[i]),
				IsStructured: true,
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func (c *CSVExcelConnector) streamExcel(ctx context.Context, out chan<- connectors.FieldRecord) error {
	f, err := excelize.OpenFile(c.path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		if len(rows) == 0 {
			continue
		}
		headers := rows[0]
		for rowIdx, row := range rows[1:] {
			for colIdx, val := range row {
				if val == "" || colIdx >= len(headers) {
					continue
				}
				select {
				case out <- connectors.FieldRecord{
					Value:        val,
					FieldName:    headers[colIdx],
					SourcePath:   fmt.Sprintf("%s:%s:row_%d.%s", c.path, sheet, rowIdx+2, headers[colIdx]),
					IsStructured: true,
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
	return nil
}

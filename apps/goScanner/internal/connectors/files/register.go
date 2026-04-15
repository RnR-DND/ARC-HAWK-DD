package files

import "github.com/arc-platform/go-scanner/internal/connectors"

func init() {
	connectors.Register("filesystem", func() connectors.Connector { return &FilesystemConnector{} })
	connectors.Register("text", func() connectors.Connector { return &TextConnector{} })
	connectors.Register("csv_excel", func() connectors.Connector { return &CSVExcelConnector{} })
	connectors.Register("html_files", func() connectors.Connector { return &HTMLConnector{} })
	connectors.Register("pdf", func() connectors.Connector { return &PDFConnector{} })
	connectors.Register("docx", func() connectors.Connector { return &DOCXConnector{} })
	connectors.Register("pptx", func() connectors.Connector { return &PPTXConnector{} })
	connectors.Register("email_files", func() connectors.Connector { return &EmailConnector{} })
	connectors.Register("avro", func() connectors.Connector { return &AvroConnector{} })
	connectors.Register("parquet", func() connectors.Connector { return &ParquetConnector{} })
	connectors.Register("orc", func() connectors.Connector { return &ORCConnector{} })
	connectors.Register("scanned_images", func() connectors.Connector { return &ImagesConnector{} })
}

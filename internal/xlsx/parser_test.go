package xlsx

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseAndValidate(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "Company Name")
	_ = f.SetCellValue(sheet, "B1", "Role")
	_ = f.SetCellValue(sheet, "C1", "Company Mail")
	_ = f.SetCellValue(sheet, "A2", "Acme")
	_ = f.SetCellValue(sheet, "B2", "Engineer")
	_ = f.SetCellValue(sheet, "C2", "hr@acme.com")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}

	result, err := ParseAndValidate(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
	if result.RowCount != 1 {
		t.Fatalf("expected 1 row, got %d", result.RowCount)
	}
}

func TestParseMissingColumn(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "Company Name")
	var buf bytes.Buffer
	_ = f.Write(&buf)

	result, err := ParseAndValidate(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid {
		t.Fatal("expected invalid result")
	}
}

func TestParseDuplicateEmail(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "Company Name")
	_ = f.SetCellValue(sheet, "B1", "Role")
	_ = f.SetCellValue(sheet, "C1", "Company Mail")
	_ = f.SetCellValue(sheet, "A2", "Acme")
	_ = f.SetCellValue(sheet, "B2", "Engineer")
	_ = f.SetCellValue(sheet, "C2", "hr@acme.com")
	_ = f.SetCellValue(sheet, "A3", "Beta")
	_ = f.SetCellValue(sheet, "B3", "Engineer")
	_ = f.SetCellValue(sheet, "C3", "hr@acme.com")

	var buf bytes.Buffer
	_ = f.Write(&buf)

	result, err := ParseAndValidate(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Fatalf("expected valid after dedup, got errors: %v", result.Errors)
	}
	if result.RowCount != 1 {
		t.Fatalf("expected 1 row, got %d", result.RowCount)
	}
	if result.Skipped.DuplicateEmail != 1 {
		t.Fatalf("expected 1 duplicate removed, got %d", result.Skipped.DuplicateEmail)
	}
}

func TestParseSkipsInvalidRowsKeepsValidOnes(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "Company Name")
	_ = f.SetCellValue(sheet, "B1", "Role")
	_ = f.SetCellValue(sheet, "C1", "Company Mail")
	_ = f.SetCellValue(sheet, "A2", "Acme")
	_ = f.SetCellValue(sheet, "B2", "Engineer")
	_ = f.SetCellValue(sheet, "C2", "hr@acme.com")
	_ = f.SetCellValue(sheet, "B3", "Designer")
	_ = f.SetCellValue(sheet, "C3", "jobs@beta.com")
	_ = f.SetCellValue(sheet, "A4", "Gamma")
	_ = f.SetCellValue(sheet, "B4", "PM")
	_ = f.SetCellValue(sheet, "C4", "not-an-email")

	var buf bytes.Buffer
	_ = f.Write(&buf)

	result, err := ParseAndValidate(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
	if result.RowCount != 1 {
		t.Fatalf("expected 1 valid row, got %d", result.RowCount)
	}
	if result.Skipped.MissingCompanyName != 1 {
		t.Fatalf("expected 1 missing company name, got %d", result.Skipped.MissingCompanyName)
	}
	if result.Skipped.InvalidEmail != 1 {
		t.Fatalf("expected 1 invalid email, got %d", result.Skipped.InvalidEmail)
	}
}

func TestParseNoValidRows(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "Company Name")
	_ = f.SetCellValue(sheet, "B1", "Role")
	_ = f.SetCellValue(sheet, "C1", "Company Mail")
	_ = f.SetCellValue(sheet, "B2", "Engineer")
	_ = f.SetCellValue(sheet, "C2", "hr@acme.com")

	var buf bytes.Buffer
	_ = f.Write(&buf)

	result, err := ParseAndValidate(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid {
		t.Fatal("expected invalid when no rows are usable")
	}
	if result.Skipped.MissingCompanyName != 1 {
		t.Fatalf("expected 1 missing company name, got %d", result.Skipped.MissingCompanyName)
	}
}

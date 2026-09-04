package xlsx

import (
	"bytes"
	"fmt"
	"net/mail"
	"strings"

	"github.com/applyflow/applyflow/internal/model"
	"github.com/xuri/excelize/v2"
)

const (
	colCompanyName = "Company Name"
	colRole        = "Role"
	colCompanyMail = "Company Mail"
	maxRows        = 400
)

type RowSkipStats struct {
	MissingCompanyName int `json:"missingCompanyName"`
	MissingRole        int `json:"missingRole"`
	MissingEmail       int `json:"missingEmail"`
	InvalidEmail       int `json:"invalidEmail"`
	DuplicateEmail     int `json:"duplicateEmail"`
	EmptyRows          int `json:"emptyRows"`
}

func (s RowSkipStats) HasSkips() bool {
	return s.MissingCompanyName > 0 ||
		s.MissingRole > 0 ||
		s.MissingEmail > 0 ||
		s.InvalidEmail > 0 ||
		s.DuplicateEmail > 0 ||
		s.EmptyRows > 0
}

type ValidationResult struct {
	Rows       []model.XLSXRow
	Valid      bool
	Errors     []string
	RowCount   int
	Skipped    RowSkipStats
	TotalRows  int
}

func ParseAndValidate(data []byte) (*ValidationResult, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid xlsx file")
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return &ValidationResult{Valid: false, Errors: []string{"no sheets found"}}, nil
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}
	if len(rows) == 0 {
		return &ValidationResult{Valid: false, Errors: []string{"empty spreadsheet"}}, nil
	}

	header := rows[0]
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	var errs []string
	for _, col := range []string{colCompanyName, colRole, colCompanyMail} {
		if _, ok := colIdx[col]; !ok {
			errs = append(errs, fmt.Sprintf("missing required column: %s", col))
		}
	}
	if len(errs) > 0 {
		return &ValidationResult{Valid: false, Errors: errs}, nil
	}

	dataRows := rows[1:]
	if len(dataRows) == 0 {
		return &ValidationResult{Valid: false, Errors: []string{"no data rows"}}, nil
	}
	if len(dataRows) > maxRows {
		return &ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("maximum %d rows allowed", maxRows)}}, nil
	}

	seen := map[string]struct{}{}
	var parsed []model.XLSXRow
	var skipped RowSkipStats

	for _, row := range dataRows {
		company := cellValue(row, colIdx[colCompanyName])
		role := cellValue(row, colIdx[colRole])
		email := strings.ToLower(strings.TrimSpace(cellValue(row, colIdx[colCompanyMail])))

		if company == "" && role == "" && email == "" {
			skipped.EmptyRows++
			continue
		}

		rowValid := true
		if company == "" {
			skipped.MissingCompanyName++
			rowValid = false
		}
		if role == "" {
			skipped.MissingRole++
			rowValid = false
		}
		if email == "" {
			skipped.MissingEmail++
			rowValid = false
		} else if !isValidEmail(email) {
			skipped.InvalidEmail++
			rowValid = false
		}

		if !rowValid {
			continue
		}

		if _, ok := seen[email]; ok {
			skipped.DuplicateEmail++
			continue
		}
		seen[email] = struct{}{}

		parsed = append(parsed, model.XLSXRow{
			CompanyName: company,
			Role:        role,
			CompanyMail: email,
		})
	}

	result := &ValidationResult{
		Rows:      parsed,
		RowCount:  len(parsed),
		Skipped:   skipped,
		TotalRows: len(dataRows),
	}

	if len(parsed) == 0 {
		result.Valid = false
		result.Errors = []string{"no valid rows found — every row was missing required data or had invalid emails"}
		return result, nil
	}

	result.Valid = true
	return result, nil
}

func cellValue(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

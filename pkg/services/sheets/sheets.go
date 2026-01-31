package sheets

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// SheetsService wraps the Google Sheets API.
type SheetsService struct {
	srv *sheets.Service
}

// New creates a new SheetsService.
func New(ctx context.Context, opts ...option.ClientOption) (*SheetsService, error) {
	srv, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Sheets client: %w", err)
	}
	return &SheetsService{srv: srv}, nil
}

// CreateSpreadsheet creates a new spreadsheet.
func (s *SheetsService) CreateSpreadsheet(title string) (*sheets.Spreadsheet, error) {
	sp := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: title,
		},
	}
	resp, err := s.srv.Spreadsheets.Create(sp).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create spreadsheet: %w", err)
	}
	return resp, nil
}

// ReadValues reads values from a range.
func (s *SheetsService) ReadValues(spreadsheetId string, rangeName string) ([][]interface{}, error) {
	resp, err := s.srv.Spreadsheets.Values.Get(spreadsheetId, rangeName).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %w", err)
	}
	return resp.Values, nil
}

// AppendValues appends values to a sheet.
// values should be a JSON string representing [][]interface{} or []interface{} (single row)
func (s *SheetsService) AppendValues(spreadsheetId string, rangeName string, valuesJSON string) (*sheets.AppendValuesResponse, error) {
	var data [][]interface{}
	
	// Try parsing as array of arrays first
	if err := json.Unmarshal([]byte(valuesJSON), &data); err != nil {
		// Try parsing as single array (single row)
		var row []interface{}
		if err2 := json.Unmarshal([]byte(valuesJSON), &row); err2 == nil {
			data = append(data, row)
		} else {
			return nil, fmt.Errorf("unable to parse values JSON: %w", err)
		}
	}

	vr := &sheets.ValueRange{
		Values: data,
	}

	// valueInputOption: USER_ENTERED allows formulas and number parsing
	resp, err := s.srv.Spreadsheets.Values.Append(spreadsheetId, rangeName, vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to append data: %w", err)
	}
	return resp, nil
}

// UpdateValues updates values in a range.
func (s *SheetsService) UpdateValues(spreadsheetId string, rangeName string, valuesJSON string) (*sheets.UpdateValuesResponse, error) {
	var data [][]interface{}
	
	if err := json.Unmarshal([]byte(valuesJSON), &data); err != nil {
		var row []interface{}
		if err2 := json.Unmarshal([]byte(valuesJSON), &row); err2 == nil {
			data = append(data, row)
		} else {
			return nil, fmt.Errorf("unable to parse values JSON: %w", err)
		}
	}

	vr := &sheets.ValueRange{
		Values: data,
	}

	resp, err := s.srv.Spreadsheets.Values.Update(spreadsheetId, rangeName, vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to update data: %w", err)
	}
	return resp, nil
}

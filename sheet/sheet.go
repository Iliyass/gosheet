package sheet

import (
	"fmt"
	"net/http"

	"gopkg.in/Iwark/spreadsheet.v2"
)

type Sheet struct {
	service *spreadsheet.Service
}

const (
	Scope = spreadsheet.Scope
)

func NewSheet(client *http.Client) (s *Sheet) {
	s = new(Sheet)
	service := spreadsheet.NewServiceWithClient(client)
	s.service = service
	return s
}

func (s *Sheet) FetchSpreadsheet(spID string) ([]map[string]interface{}, error) {
	sp, err := s.service.FetchSpreadsheet(spID)
	if err != nil {
		return nil, fmt.Errorf("Error in fetching Spreadsheet %v, err: %v", spID, err)
	}

	d := []map[string]interface{}{}
	for _, r := range sp.Sheets[0].Rows[1:] {
		spr := map[string]interface{}{}
		for _, rh := range sp.Sheets[0].Rows[0] {
			spr[rh.Value] = r[rh.Column].Value
		}
		d = append(d, spr)
	}
	return d, nil
}

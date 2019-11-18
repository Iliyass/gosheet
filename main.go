package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2/google"
	"gopkg.in/Iwark/spreadsheet.v2"
)

func connect() *spreadsheet.Service {
	data, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		panic(err)
	}
	conf, err := google.JWTConfigFromJSON(data, spreadsheet.Scope)
	if err != nil {
		panic(err)
	}
	client := conf.Client(context.TODO())
	service := spreadsheet.NewServiceWithClient(client)
	return service
}

func handleFetch(w http.ResponseWriter, req *http.Request) {
	s := connect()
	spreadsheetID := "18pD5O5jXJgDBp_V7mbI1rUgeNPcyFqom6FXJOT4sRI0"
	spreadsheet, err := s.FetchSpreadsheet(spreadsheetID)
	if err != nil {
		panic(err)
	}
	sp := []map[string]interface{}{}
	for _, r := range spreadsheet.Sheets[0].Rows[1:] {
		spr := map[string]interface{}{}
		for _, rh := range spreadsheet.Sheets[0].Rows[0] {
			spr[rh.Value] = r[rh.Column].Value
		}
		sp = append(sp, spr)
	}
	data, err := json.Marshal(sp)
	w.Write([]byte(data))
}
func main() {
	http.HandleFunc("/", handleFetch)
	http.ListenAndServe(":3009", nil)
}

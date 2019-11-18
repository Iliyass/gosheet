package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"../sheet-api/sheet"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var config *oauth2.Config

// init is invoked before main()
func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	config = &oauth2.Config{
		ClientID:     getEnv("GOOGLE_OAUTH_CLIENTID"),
		ClientSecret: getEnv("GOOGLE_OAUTH_CLIENTSECRET"),
		Endpoint:     google.Endpoint,
		RedirectURL:  getEnv("GOOGLE_OAUTH_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/spreadsheets.readonly",
			"https://www.googleapis.com/auth/spreadsheets",
			sheet.Scope,
		},
	}
}

// Simple helper function to read an environment or return a default value
func getEnv(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	panic(fmt.Errorf("Env Variable is not defined %v", key))
}

func handleOAuth(w http.ResponseWriter, req *http.Request) {
	code := req.FormValue("code")
	ctx := context.Background()

	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Println(err.Error())
		http.Redirect(w, req, "/", http.StatusTemporaryRedirect)
	}
	client := config.Client(ctx, token)

	s := sheet.NewSheet(client)
	spreadsheetID := "19XI3VcIWi5UqPCL4FTotcZdIQAngjGiH3fLWzom59P8"
	spreadsheet, err := s.FetchSpreadsheet(spreadsheetID)
	if err != nil {
		panic(err)
	}

	data, err := json.Marshal(spreadsheet)
	w.Write([]byte(data))
}

func handleConnect(w http.ResponseWriter, req *http.Request) {
	url := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, req, url, 301)
}

func main() {
	var port string
	flag.StringVar(&port, "port", "3009", "port server")
	flag.Parse()
	http.HandleFunc("/", handleConnect)
	http.HandleFunc("/oatuh", handleOAuth)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

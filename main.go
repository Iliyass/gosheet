package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"../sheet-api/sheet"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
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
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/drive.file",
			"https://www.googleapis.com/auth/drive.readonly",
			"https://www.googleapis.com/auth/drive.metadata.readonly",
			"https://www.googleapis.com/auth/drive.appdata",
			"https://www.googleapis.com/auth/drive.metadata",
			"https://www.googleapis.com/auth/drive.photos.readonly",
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

func fetchUserInfo(client *http.Client) (*sheet.GoogleUser, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result sheet.GoogleUser
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
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
	userData, err := fetchUserInfo(client)
	userData.AccessToken = token.AccessToken
	userData.RefreshToken = token.RefreshToken

	user, err := sheet.CreateOrGetUser(userData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	data, err := json.Marshal(user)
	w.Write([]byte(data))
}

func handleConnect(w http.ResponseWriter, req *http.Request) {
	url := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, req, url, http.StatusTemporaryRedirect)
}

func parseSheet(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	query := req.URL.Query()
	fileName, ok := query["file"]
	fmt.Println(fileName)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
	user, err := sheet.GetUser("psycho.ilyass@gmail.com")
	if err != nil {
		http.Error(w, "user not found", http.StatusInternalServerError)
	}
	token := new(oauth2.Token)
	token.RefreshToken = user.RefreshToken
	client := config.Client(ctx, token)
	srv, err := drive.New(client)
	r, err := srv.Files.List().PageSize(10).Fields("nextPageToken, files(id, name)").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}
	fmt.Println("Files:")
	if len(r.Files) == 0 {
		fmt.Println("No files found.")
	} else {
		for _, i := range r.Files {
			fmt.Println(i.Name, fmt.Sprint(fileName))
			if i.Name == fileName[0] {
				s := sheet.NewSheet(client)
				sheetData, err := s.FetchSpreadsheet(i.Id)
				if err != nil {
					http.Error(w, "Something went wrong in parsing", http.StatusNotFound)
				}
				data, err := json.Marshal(sheetData)
				w.Write([]byte(data))
			}
		}
	}
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}
}

func main() {
	var port string
	flag.StringVar(&port, "port", "3009", "port server")
	flag.Parse()
	http.HandleFunc("/login", handleConnect)
	http.HandleFunc("/oauth", handleOAuth)
	http.HandleFunc("/parse", parseSheet)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

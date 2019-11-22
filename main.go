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
	"strconv"

	"../sheet-api/sheet"
	"github.com/HouzuoGuo/tiedot/db"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const dbPath = "./db"

var config *oauth2.Config
var myDB *db.DB

type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AccessToken string `json:"access_token"`
}

func NewUser(u map[string]interface{}) (*User, error) {
	if _, ok := u["id"]; !ok {
		return nil, fmt.Errorf("ID is required")
	}
	if _, ok := u["name"]; !ok {
		return nil, fmt.Errorf("Name is required")
	}
	if _, ok := u["email"]; !ok {
		return nil, fmt.Errorf("Name is required")
	}
	if _, ok := u["access_token"]; !ok {
		return nil, fmt.Errorf("Access Token is required")
	}
	return &User{
		ID:          u["id"].(string),
		Name:        u["name"].(string),
		Email:       u["email"].(string),
		AccessToken: u["access_token"].(string),
	}, nil
}

type GoogleUser struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Profile       string `json:"profile"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Gender        string `json:"gender"`
	AccessToken   string `json:"access_token"`
}

func (g *GoogleUser) toMap() map[string]interface{} {
	return map[string]interface{}{
		"name":         g.Name,
		"email":        g.Email,
		"access_token": g.AccessToken,
	}
}

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
			sheet.Scope,
		},
	}
	myDB, err := db.OpenDB(dbPath)
	if err != nil {
		panic(err)
	}

	err = myDB.Drop("Users")
	if !myDB.ColExists("Users") {
		if err := myDB.Create("Users"); err != nil {
			panic(err)
		}
	}
}

// Simple helper function to read an environment or return a default value
func getEnv(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	panic(fmt.Errorf("Env Variable is not defined %v", key))
}

func fetchUserInfo(client *http.Client) (*GoogleUser, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result GoogleUser
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func createOrGetUser(userData *GoogleUser) (*User, error) {
	myDB, err := db.OpenDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("Can't Open Database Error: %v", err)
	}
	if !myDB.ColExists("Users") {
		if err := myDB.Create("Users"); err != nil {
			return nil, fmt.Errorf("Can't Create Users Collection Error: %v", err)
		}
	}
	users := myDB.Use("Users")
	if err := users.Index([]string{"email"}); err != nil {
		return nil, fmt.Errorf("Can't create an email index: %v", err)
	}
	var query interface{}
	q := fmt.Sprintf(`[{"eq": "%v", "in": ["email"]}]`, userData.Email)
	log.Println("Query: ", q)
	json.Unmarshal([]byte(q), &query)
	queryResult := make(map[int]struct{})
	if err := db.EvalQuery(query, users, &queryResult); err != nil {
		return nil, fmt.Errorf("Executing query has failed: %v", err)
	}
	// Query result are document IDs
	for id := range queryResult {
		// To get query result document, simply read it
		readBack, err := users.Read(id)
		if err != nil {
			return nil, fmt.Errorf("Reading users with id has failed: %v", err)
		}
		log.Println("Got queryResult", readBack)
		return NewUser(readBack)
	}
	userID, err := users.Insert(userData.toMap())
	if err != nil {
		return nil, fmt.Errorf("Cannot insert user: %v", err)
	}
	log.Println("User saved - User Id:", userID)
	// Read document
	readBack, err := users.Read(userID)
	if err != nil {
		return nil, fmt.Errorf("Cannot Read user: %v", err)
	}
	if err := users.Unindex([]string{"email"}); err != nil {
		panic(err)
	}
	readBack["id"] = strconv.Itoa(userID)
	log.Println("User retried - User:", readBack)
	return NewUser(readBack)
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

	user, err := createOrGetUser(userData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	data, err := json.Marshal(user)
	w.Write([]byte(data))

	// s := sheet.NewSheet(client)
	// spreadsheetID := "19XI3VcIWi5UqPCL4FTotcZdIQAngjGiH3fLWzom59P8"
	// spreadsheet, err := s.FetchSpreadsheet(spreadsheetID)
	// if err != nil {
	// 	panic(err)
	// }

	// data, err := json.Marshal(spreadsheet)
	// w.Write([]byte(data))
}

func handleConnect(w http.ResponseWriter, req *http.Request) {
	url := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, req, url, http.StatusTemporaryRedirect)
}

func main() {
	var port string
	flag.StringVar(&port, "port", "3009", "port server")
	flag.Parse()
	http.HandleFunc("/login", handleConnect)
	http.HandleFunc("/oauth", handleOAuth)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

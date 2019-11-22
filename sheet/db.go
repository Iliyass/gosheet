package sheet

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/HouzuoGuo/tiedot/db"
)

const dbPath = "./db"

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

type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AccessToken string `json:"access_token"`
}

func NewUser(u map[string]interface{}) (*User, error) {
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
		Name:        u["name"].(string),
		Email:       u["email"].(string),
		AccessToken: u["access_token"].(string),
	}, nil
}

func CreateOrGetUser(userData *GoogleUser) (*User, error) {
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
		if err.Error() != "Path [email] is already indexed" {
			return nil, fmt.Errorf("Can't create an email index: %v", err)
		}
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

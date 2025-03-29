package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	ACCESS_TOKEN = "ACCESS_TOKEN"
	DATASET      = "dataset.xml"
)

type TestServer struct {
	Server *httptest.Server
	Client SearchClient
}

func InitServer(token string) *TestServer {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{token, server.URL}
	return &TestServer{
		Server: server,
		Client: client,
	}
}

func (t *TestServer) Close() {
	t.Server.Close()
}

type XMLData struct {
	XMLName xml.Name `xml:"root"`
	XMLRows []XMLRow `xml:"row"`
}

type XMLRow struct {
	XMLName   xml.Name `xml:"row"`
	Id        int      `xml:"id"`
	FirstName string   `xml:"first_name"`
	LastName  string   `xml:"last_name"`
	Age       int      `xml:"age"`
	About     string   `xml:"about"`
	Gender    string   `xml:"gender"`
}

func SendError(w http.ResponseWriter, str string, status int) {
	js, err := json.Marshal(SearchErrorResponse{str})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintln(w, string(js))
}

func SearchServer(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("AccessToken") != ACCESS_TOKEN {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	dataXML, _ := os.Open(DATASET)

	var (
		data  XMLData
		users []User
	)
	d, _ := io.ReadAll(dataXML)
	err := xml.Unmarshal(d, &data)
	if err != nil {
		http.Error(w, "Error in xml marshaling", http.StatusBadRequest)
		return
	}

	defer dataXML.Close()

	r := req.URL.Query()

	query := r.Get("query")
	for _, row := range data.XMLRows {
		if query != "" {
			flag := strings.Contains(row.FirstName, query) || strings.Contains(row.LastName, query) ||
				strings.Contains(row.About, query)
			if !flag {
				continue
			}
		}
		users = append(users, User{
			Id:     row.Id,
			Name:   row.FirstName + " " + row.LastName,
			Age:    row.Age,
			About:  row.About,
			Gender: row.Gender,
		})
	}
	orderBy, _ := strconv.Atoi(r.Get("order_by"))
	if orderBy < -1 || orderBy > 1 {
		http.Error(w, "Incorrect order", http.StatusBadRequest)
		return
	}
	if orderBy != OrderByAsIs {
		var cmp func(u1, u2 User) bool
		switch r.Get("order_field") {
		case "Id":
			cmp = func(u1, u2 User) bool {
				return u1.Id > u2.Id
			}
		case "Name":
			cmp = func(u1, u2 User) bool {
				return u1.Name > u2.Name
			}
		case "":
			cmp = func(u1, u2 User) bool {
				return u1.Name > u2.Name
			}
		case "Age":
			cmp = func(u1, u2 User) bool {
				return u1.Age > u2.Age
			}
		default:
			SendError(w, "OrderField is invalid", http.StatusBadRequest)
			return
		}
		sort.Slice(users, func(i, j int) bool {
			return cmp(users[i], users[j]) && orderBy == OrderByDesc
		})
	}

	limit, _ := strconv.Atoi(r.Get("limit"))
	offset, _ := strconv.Atoi(r.Get("offset"))
	limit--
	if limit > len(users) {
		http.Error(w, "Incorrect up limit", http.StatusBadRequest)
		return
	} else if limit < 0 {
		http.Error(w, "Impossible have a negative limit", http.StatusBadRequest)
		return
	}

	if offset < 0 {
		http.Error(w, "Sequence can't start with negative position", http.StatusBadRequest)
		return
	} else if offset > len(users) {
		http.Error(w, "We can't miss so much elements", http.StatusBadRequest)
		return
	}

	if offset+limit > len(users) {
		http.Error(w, "Ending position behind the slice", http.StatusBadRequest)
		return
	}

	if limit > 0 {
		from := offset
		if from > len(users)-1 {
			users = []User{}
		} else {
			to := offset + limit
			if to > len(users) {
				to = len(users)
			}

			users = users[from:to]
		}
	}
	marshal, err := json.Marshal(users)
	if err != nil {
		http.Error(w, "Error in json marshaling", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(marshal)
}

func TestLimitLow(t *testing.T) {
	ts := InitServer(ACCESS_TOKEN)
	defer ts.Close()

	_, err := ts.Client.FindUsers(SearchRequest{
		Limit: -1,
	})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "limit must be > 0" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestLimitHigh(t *testing.T) {
	ts := InitServer(ACCESS_TOKEN)
	defer ts.Close()

	response, _ := ts.Client.FindUsers(SearchRequest{
		Limit: 100,
	})

	if len(response.Users) != 25 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
	}
}

func TestInvalidToken(t *testing.T) {
	ts := InitServer(ACCESS_TOKEN + "123")
	defer ts.Close()

	_, err := ts.Client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	}
	if err != nil && err.Error() != "Bad AccessToken" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestInvalidOrderField(t *testing.T) {
	ts := InitServer(ACCESS_TOKEN)
	defer ts.Close()

	_, err := ts.Client.FindUsers(SearchRequest{
		OrderBy:    OrderByAsc,
		OrderField: "Foo",
	})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "unknown bad request error: OrderField is invalid" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestOffsetLow(t *testing.T) {
	ts := InitServer(ACCESS_TOKEN)
	defer ts.Close()

	_, err := ts.Client.FindUsers(SearchRequest{
		Offset: -1,
	})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "offset must be > 0" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestFindUserByName(t *testing.T) {
	ts := InitServer(ACCESS_TOKEN)
	defer ts.Close()

	response, _ := ts.Client.FindUsers(SearchRequest{
		Query: "Annie",
		Limit: 1,
	})

	if len(response.Users) != 1 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
		return
	}

	if response.Users[0].Name != "Annie Osborn" {
		t.Errorf("Invalid user found: %v", response.Users[0])
		return
	}
}

func TestLimitOffset(t *testing.T) {
	ts := InitServer(ACCESS_TOKEN)
	defer ts.Close()

	response, _ := ts.Client.FindUsers(SearchRequest{
		Limit:  3,
		Offset: 0,
	})

	if len(response.Users) != 3 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
		return
	}

	if response.Users[2].Name != "Brooks Aguilar" {
		t.Errorf("Invalid user at position 3: %v", response.Users[2])
		return
	}

	response, _ = ts.Client.FindUsers(SearchRequest{
		Limit:  5,
		Offset: 2,
	})

	if len(response.Users) != 5 {
		t.Errorf("Invalid number of users: %d", len(response.Users))
		return
	}

	if response.Users[0].Name != "Brooks Aguilar" {
		t.Errorf("Invalid user at position 3: %v", response.Users[0])
		return
	}
}

func TestFatalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Fatal Error", http.StatusInternalServerError)
	}))
	client := SearchClient{ACCESS_TOKEN, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if err.Error() != "SearchServer fatal error" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestCantUnpackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Some Error", http.StatusBadRequest)
	}))
	client := SearchClient{ACCESS_TOKEN, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "cant unpack error json") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnknownBadRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SendError(w, "Unknown Error", http.StatusBadRequest)
	}))
	client := SearchClient{ACCESS_TOKEN, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "unknown bad request error") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestCantUnpackResultError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "None")
	}))
	client := SearchClient{ACCESS_TOKEN, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	client := SearchClient{ACCESS_TOKEN, server.URL}
	defer server.Close()

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if !strings.Contains(err.Error(), "timeout for") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnknownError(t *testing.T) {
	client := SearchClient{ACCESS_TOKEN, "http://invalid-server/"}

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("Empty error")
	} else if strings.Contains(err.Error(), "unknown error") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

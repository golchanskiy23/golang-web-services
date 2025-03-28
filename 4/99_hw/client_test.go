package main

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	AccessToken = "123"
)

type TestServer struct {
	Server *httptest.Server
	Client *SearchClient
}

type XMLData struct {
	Name    string   `xml:"name"`
	XMLRows []XMLRow `xml:"rows"`
}

type XMLRow struct {
	Id        int    `xml:"id"`
	Age       int    `xml:"age"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Gender    string `xml:"gender"`
	About     string `xml:"about"`
}

func SearchServer(w http.ResponseWriter, req *http.Request) {
	// проверка авторизации
	if req.Header.Get("AccessToken") != AccessToken {
		http.Error(w, "Authorization failed", http.StatusUnauthorized)
		return
	}

	// открываем xml-ресурс
	xmlData, err := os.Open("dataset.xml")
	defer xmlData.Close()

	if err != nil {
		http.Error(w, "Error opening dataset.xml", http.StatusBadRequest)
		return
	}

	// вытягиваем все данные из xml
	b, err := io.ReadAll(xmlData)
	if err != nil {
		http.Error(w, "Error reading dataset.xml", http.StatusBadRequest)
	}

	var (
		data  XMLData
		users []User
	)

	xml.Unmarshal(b, &data)

	r := req.URL.Query()

	// собираем всех юзеров из файла
	resp := r.Get("query")
	for _, row := range data.XMLRows {
		if resp != "" {
			isNotEmpty := strings.Contains(row.FirstName, resp) ||
				strings.Contains(row.LastName, resp) || strings.Contains(row.About, resp)

			if isNotEmpty {
				users = append(users, User{
					Id:     row.Id,
					Name:   row.FirstName + " " + row.LastName,
					Gender: row.Gender,
					About:  row.About,
				})
			}
		}
	}

	orderById, _ := strconv.Atoi(r.Get("order_by"))
	if orderById != 0 {
		var cmp func(t1, t2 User) bool
		switch r.Get("order_field") {
		case "Id":
			cmp = func(t1, t2 User) bool {
				return t1.Id < t2.Id
			}
		case "Age":
			cmp = func(t1, t2 User) bool {
				return t1.Age < t2.Age
			}
		case "Name":
			cmp = func(t1, t2 User) bool {
				return t1.Name < t2.Name
			}
		case "":
			cmp = func(t1, t2 User) bool {
				return t1.Name < t2.Name
			}
		default:
			http.Error(w, ErrorBadOrderField, http.StatusBadRequest)
			return
		}
		sort.Slice(users, func(i, j int) bool {
			return cmp(users[i], users[j]) && orderById == OrderByAsc
		})
	}

	limit, _ := strconv.Atoi(r.Get("limit"))
	offset, _ := strconv.Atoi(r.Get("offset"))
	if limit > 0 {
		from, to := offset, offset+limit
		if to > len(users) {
			http.Error(w, "Incorrect limit and offset combination", http.StatusBadRequest)
			return
		}
		users = users[from:to]
	}

	bytesSlices, _ := xml.Marshal(&users)
	w.Header().Set("Content-Type", "application/xml")
	w.Write(bytesSlices)
}

func InitServer(token string) *TestServer {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := &SearchClient{token, server.URL}
	return &TestServer{server, client}
}

func Test(t *testing.T) {
}

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type DbExplorer struct {
	db     *sql.DB
	Tables map[string]Table
}

type Response struct {
	Data interface{} `json:"data"`
}

type Table struct {
}

type Request struct {
	Req       *http.Request
	Table     *Table
	RequestID int
}

type ResponseError struct {
	ErrName string `json:"error"`
	Code    int    `json:"code"`
}

func (rerror ResponseError) Error() string {
	return rerror.ErrName
}

func (e *DbExplorer) CreateNewRequest(r *http.Request) (*Request, error) {
	ans := &Request{Req: r, RequestID: -1}
	if r.URL.Path == "/" {
		return ans, nil
	}
	seps := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	size := len(seps)
	if size >= 1 {
		if t, ok := e.Tables[seps[0]]; ok {
			ans.Table = &t
		} else {
			return nil, ResponseError{"unknown table", http.StatusNotFound}
		}
	}

	if size >= 2 {
		if id, err := strconv.Atoi(seps[1]); err == nil {
			ans.RequestID = id
		} else {
			return nil, ResponseError{"unknown id", http.StatusNotFound}
		}
	}
	return ans, nil
}

func (e *DbExplorer) HandleRequest(r *http.Request) (interface{}, error) {
	request, err := e.CreateNewRequest(r)
	if err != nil {
		return nil, err
	}
	switch r.Method {
	case http.MethodPost:
		if request.Table != nil && request.RequestID != -1 {
			data, err_ := request.GetData()
			if err_ != nil {
				return nil, err_
			}

			return e.HandlePostDataTable(request.Table, request.RequestID, data)
		}
	case http.MethodGet:
		if request.Table == nil {
			return e.HandleGetTables()
		}
		if request.RequestID == -1 {
			limit, offset := request.GetLimitOffset()
			return e.HandleLimitOffset(request.Table, limit, offset)
		}
	case http.MethodDelete:
		if request.Table != nil && request.RequestID != -1 {
			return e.HandleDeletePost(request.Table, request.RequestID)
		}
	case http.MethodPut:
		if request.Table != nil {
			data, err_ := request.GetData()
			if err_ != nil {
				return nil, err_
			}

			return e.HandlePutData(request.Table, data)
		}
	}
	return nil, ResponseError{"unknown method", http.StatusMethodNotAllowed}
}

func (e *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := Response{}
	data, err := e.HandleRequest(r)
	if err != nil {
		if re, ok := err.(ResponseError); ok {
			w.WriteHeader(re.Code)
		}
	} else {
		res.Data = data
	}
	ans, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "Something error during marshalling", http.StatusBadRequest)
	}
	w.Write(ans)
}

func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	dbExplorer := &DbExplorer{db: db}
	tables, err := dbExplorer.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to create DbExplorer: %s", err)
	}
	return &DbExplorer{db: db, Tables: tables}, nil
}

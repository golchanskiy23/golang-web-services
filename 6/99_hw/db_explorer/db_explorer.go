package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type DbExplorer struct {
	db *sql.DB
}

type Result struct {
	Data interface{} `json:"data"`
}

// парсинг исходного запроса и выделение нужных компонент
func (e *DbExplorer) HandleRequest(r *http.Request) (interface{}, error) {
	return Result{}, nil
}

func (e *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := Result{}
	data, err := e.HandleRequest(r)
	if err != nil {
		http.Error(w, "Something error during request  handling", http.StatusInternalServerError)
	} else {
		res.Data = data
	}
	ans, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "Something error during request  handling 2", http.StatusInternalServerError)
	}
	w.Write(ans)
}

func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	return &DbExplorer{db: db}, nil
}

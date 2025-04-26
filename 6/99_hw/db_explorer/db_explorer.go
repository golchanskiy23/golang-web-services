package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const defaultLimit int = 5

type DbExplorer struct {
	db     *sql.DB
	Tables map[string]Table
}

type Response struct {
	Data interface{}
}

type Table struct {
	TableColumns []TableColumn
}

type TableColumn struct {
	Field      string
	Type       ColumnFiller
	Collation  interface{}
	Null       bool
	Key        string
	Default    interface{}
	Extra      string
	Privileges string
	Comment    string
}

type Request struct {
	Req       *http.Request
	Table     *Table
	RequestID int
}

type ResponseError struct {
	ErrName string
	Code    int
}

func (r ResponseError) Error() string {
	return r.ErrName
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
			return nil, ResponseError{"unknown table", http.StatusInternalServerError}
		}
	}

	if size >= 2 {
		if id, err := strconv.Atoi(seps[1]); err == nil {
			ans.RequestID = id
		} else {
			return nil, ResponseError{"unknown id", http.StatusInternalServerError}
		}
	}
	return ans, nil
}

type BodyRecord struct {
	Data []byte
}

// придумать структуры возвращаемых значений для каждого запроса
func (req *Request) GetData() (interface{}, error) {
	data, err := io.ReadAll(req.Req.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading request body: %v", err)
	}
	var record BodyRecord
	json.Unmarshal(data, &record)
	return record, nil
}

func (req *Request) GetLimitOffset() (int, int) {
	query := req.Req.URL.Query()
	var (
		limit, offset int
		err           error
	)
	if strLimit := query.Get("limit"); strLimit != "" {
		limit, err = strconv.Atoi(strLimit)
		if err != nil {
			limit = defaultLimit
		}
	}

	if strOffset := query.Get("offset"); strOffset != "" {
		offset, _ = strconv.Atoi(strOffset)
	}
	return limit, offset
}

func (e *DbExplorer) HandlePostDataTable(t Table, id int, data interface{}) (interface{}, error) {
	return nil, nil
}

func (e *DbExplorer) HandleGetTables() (interface{}, error) {
	return nil, nil
}

func (e *DbExplorer) HandleLimitOffset(t Table, limit int, offset int) (interface{}, error) {
	return nil, nil
}

func (e *DbExplorer) HandlePutData(t Table, data interface{}) (interface{}, error) {
	return nil, nil
}

func (e *DbExplorer) HandleDeletePost(t Table, id int) (interface{}, error) {
	return nil, nil
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

			return e.HandlePostDataTable(*request.Table, request.RequestID, data)
		}
	case http.MethodGet:
		if request.Table == nil {
			return e.HandleGetTables()
		}
		if request.RequestID == -1 {
			limit, offset := request.GetLimitOffset()
			return e.HandleLimitOffset(*request.Table, limit, offset)
		}
	case http.MethodDelete:
		if request.Table != nil && request.RequestID != -1 {
			return e.HandleDeletePost(*request.Table, request.RequestID)
		}
	case http.MethodPut:
		if request.Table != nil {
			data, err_ := request.GetData()
			if err_ != nil {
				return nil, err_
			}

			return e.HandlePutData(*request.Table, data)
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

func (e *DbExplorer) GetTablesNames() ([]string, error) {
	tables, err := e.db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer tables.Close()
	tablesNames := make([]string, 0)
	var s string
	for tables.Next() {
		tables.Scan(&s)
		tablesNames = append(tablesNames, s)
	}
	return tablesNames, nil
}

type ColumnFiller interface {
	NewFiller() interface{}
}

type IntColumn struct {
	Flag bool
}

func (c IntColumn) NewFiller() interface{} {
	if c.Flag {
		return new(*int64)
	}
	return new(int64)
}

type StringColumn struct {
	Flag bool
}

func (c StringColumn) NewFiller() interface{} {
	if c.Flag {
		return new(*string)
	}
	return new(string)
}

func (e *DbExplorer) GetTableColumns(name string) ([]TableColumn, error) {
	columns, err := e.db.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", name))
	if err != nil {
		return nil, fmt.Errorf("error in the sqlQuery executing: %s", err)
	}
	defer columns.Close()
	var (
		colType_ string
		colNull  string
		isNull   bool
	)
	colSlice := make([]TableColumn, 0)
	for columns.Next() {
		col := TableColumn{}
		columns.Scan(
			&col.Field,
			&colType_,
			&col.Collation,
			&colNull,
			&col.Key,
			&col.Default,
			&col.Extra,
			&col.Privileges,
			&col.Comment,
		)
		isNull = colNull == "YES"
		if strings.Contains(colType_, "int") {
			col.Type = IntColumn{isNull}
		} else {
			col.Type = StringColumn{isNull}
		}
		colSlice = append(colSlice, col)
	}
	return colSlice, nil
}

func (e *DbExplorer) GetTables() (map[string]Table, error) {
	names, err := e.GetTablesNames()
	if err != nil {
		return nil, fmt.Errorf("error in the receiving of all tables: %s", err)
	}

	resultMap := map[string]Table{}
	for _, name := range names {
		columns, err_ := e.GetTableColumns(name)
		if err_ != nil {
			return nil, fmt.Errorf("error in the receiving of table's columns: %s", err)
		}
		currTable := Table{
			TableColumns: columns,
		}
		resultMap[name] = currTable
	}

	return resultMap, nil
}

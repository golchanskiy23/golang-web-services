package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const defaultLimit = 5

type IntColumn struct {
	Null bool
}

type StringColumn struct {
	Null bool
}

type ColumnType interface {
	NewVar() interface{}
	IsValidValue(val interface{}) bool
}

func (c IntColumn) NewVar() interface{} {
	if c.Null {
		return new(*int64)
	} else {
		return new(int64)
	}
}

func (c IntColumn) IsValidValue(val interface{}) bool {
	if val == nil {
		return c.Null
	}

	_, ok := val.(int64)
	return ok
}

func (c StringColumn) NewVar() interface{} {
	if c.Null {
		return new(*string)
	} else {
		return new(string)
	}
}

func (c StringColumn) IsValidValue(val interface{}) bool {
	if val == nil {
		return c.Null
	}

	_, ok := val.(string)
	return ok
}

type TableColumn struct {
	Field      string
	Type       ColumnType
	Collation  interface{}
	Null       bool
	Key        string
	Default    interface{}
	Extra      string
	Privileges string
	Comment    string
}

type Table struct {
	Name    string
	Pk      string
	Columns []TableColumn
}

type DbExplorer struct {
	db     *sql.DB
	tables map[string]Table
}

func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	dbExplorer := &DbExplorer{db: db}
	tables, err := dbExplorer.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to create DbExplorer: %s", err)
	}
	dbExplorer.tables = tables
	return dbExplorer, nil
}

func (d *DbExplorer) GetTables() (map[string]Table, error) {
	names, err := d.GetTableNames()
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %s", err)
	}

	tables := make(map[string]Table, len(names))
	for _, name := range names {
		columns, err := d.GetTableColumns(name)
		if err != nil {
			return nil, fmt.Errorf("failed to get tables: %s", err)
		}

		table := Table{
			Name:    name,
			Columns: columns,
		}

		for _, col := range columns {
			if col.Key == "PRI" {
				table.Pk = col.Field
				break
			}
		}
		tables[name] = table
	}
	return tables, nil
}

func (d *DbExplorer) GetTableNames() (tables []string, err error) {
	rows, err := d.db.Query("SHOW TABLES")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch table names: %s", err)
	}
	defer rows.Close()

	var t string
	for rows.Next() {
		rows.Scan(&t)
		tables = append(tables, t)
	}

	return
}

func (d *DbExplorer) GetTableColumns(table string) (columns []TableColumn, err error) {
	rows, err := d.db.Query("SHOW FULL COLUMNS FROM " + table)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch columns for table '%s': %s", table, err)
	}
	defer rows.Close()

	var (
		colType string
		colNull string
		isNull  bool
	)

	for rows.Next() {
		col := TableColumn{}
		rows.Scan(
			&col.Field,
			&colType,
			&col.Collation,
			&colNull,
			&col.Key,
			&col.Default,
			&col.Extra,
			&col.Privileges,
			&col.Comment,
		)

		isNull = colNull == "YES"
		if strings.Contains(colType, "int") {
			col.Type = IntColumn{isNull}
		} else {
			col.Type = StringColumn{isNull}
		}

		columns = append(columns, col)
	}

	return
}

type Response struct {
	Data  interface{} `json:"response,omitempty"`
	Error string      `json:"error,omitempty"`
}

type ResponseError struct {
	Text       string
	StatusCode int
}

func (e ResponseError) Error() string {
	return e.Text
}

type Request struct {
	request  *http.Request
	Table    *Table
	RecordId *int
}

func (e *DbExplorer) newRequest(r *http.Request) (*Request, error) {
	req := &Request{
		request: r,
	}

	if r.URL.Path == "/" {
		return req, nil
	}

	urlParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(urlParts) >= 1 {
		if t, ok := e.tables[urlParts[0]]; ok {
			req.Table = &t
		} else {
			return nil, ResponseError{"unknown table", http.StatusNotFound}
		}
	}

	if len(urlParts) >= 2 {
		if id, err := strconv.Atoi(urlParts[1]); err == nil {
			req.RecordId = &id
		}
	}

	return req, nil
}

type GetTablesResponse struct {
	Tables []string `json:"tables"`
}

func (e *DbExplorer) handleGetTables() (*GetTablesResponse, error) {
	tables := make([]string, 0, len(e.tables))
	for table, _ := range e.tables {
		tables = append(tables, table)
	}

	sort.Strings(tables)

	return &GetTablesResponse{
		Tables: tables,
	}, nil
}

func (r *Request) GetLimitOffset() (limit, offset int) {
	var err error
	q := r.request.URL.Query()

	limit = defaultLimit
	if limitStr := q.Get("limit"); limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			limit = defaultLimit
		}
	}

	if offsetStr := q.Get("offset"); offsetStr != "" {
		offset, _ = strconv.Atoi(offsetStr)
	}

	return
}

type TableRecord map[string]interface{}

type GetTableRecordsResponse struct {
	Records []TableRecord `json:"records"`
}

type TableRow []interface{}

func (t *Table) NewRow() TableRow {
	row := make(TableRow, len(t.Columns))
	for i := range row {
		row[i] = t.Columns[i].Type.NewVar()
	}

	return row
}

func (t *Table) NewRecord(row TableRow) TableRecord {
	record := TableRecord{}
	for i, c := range t.Columns {
		record[c.Field] = row[i]
	}

	return record
}

func (e *DbExplorer) handleGetTableRecords(table Table, limit int, offset int) (*GetTableRecordsResponse, error) {
	q := fmt.Sprintf("SELECT * FROM %s  LIMIT ? OFFSET ?", table.Name)
	rows, err := e.db.Query(q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []TableRecord
	for rows.Next() {
		row := table.NewRow()
		if err := rows.Scan(row...); err != nil {
			return nil, err
		}

		records = append(records, table.NewRecord(row))
	}

	return &GetTableRecordsResponse{
		Records: records,
	}, nil
}

type GetTableRecordResponse struct {
	Record TableRecord `json:"record"`
}

func (e *DbExplorer) handleGetTableRecord(table Table, id int) (*GetTableRecordResponse, error) {
	q := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", table.Name, table.Pk)
	row := e.db.QueryRow(q, id)

	r := table.NewRow()
	if err := row.Scan(r...); err != nil {
		return nil, ResponseError{"record not found", http.StatusNotFound}
	}

	return &GetTableRecordResponse{
		Record: table.NewRecord(r),
	}, nil
}

func (r *Request) GetRecordData() (record TableRecord, err error) {
	body, err := ioutil.ReadAll(r.request.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &record)
	return
}

type PutTableRecordResponse map[string]int

func (e *DbExplorer) handlePutTableRecord(table Table, data TableRecord) (*PutTableRecordResponse, error) {
	var (
		inCols []string
		inVals []interface{}
	)

	for _, col := range table.Columns {
		if col.Field == table.Pk {
			continue
		}

		inCols = append(inCols, col.Field)
		if val, ok := data[col.Field]; ok {
			inVals = append(inVals, val)
		} else {
			inVals = append(inVals, col.Type.NewVar())
		}
	}

	q := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table.Name,
		strings.Join(inCols, ", "),
		strings.Join(strings.Split(strings.Repeat("?", len(inCols)), ""), ", "),
	)

	res, err := e.db.Exec(q, inVals...)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &PutTableRecordResponse{
		table.Pk: int(id),
	}, nil
}

type PostTableRecordResponse struct {
	Updated int `json:"updated"`
}

func (t *Table) ValidateRecord(record TableRecord) error {
	for _, c := range t.Columns {
		if v, ok := record[c.Field]; ok {
			if c.Field == t.Pk || !c.Type.IsValidValue(v) {
				return NewValidationError(c.Field)
			}
		}
	}

	return nil
}

func NewValidationError(field string) ResponseError {
	return ResponseError{
		Text:       fmt.Sprintf("field %s have invalid type", field),
		StatusCode: http.StatusBadRequest,
	}
}

func (e *DbExplorer) handlePostTableRecord(table Table, id int, data TableRecord) (*PostTableRecordResponse, error) {
	if err := table.ValidateRecord(data); err != nil {
		return nil, err
	}

	var (
		uSets []string
		uVals []interface{}
	)

	for k, v := range data {
		uSets = append(uSets, fmt.Sprintf("%s = ?", k))
		uVals = append(uVals, v)
	}
	uVals = append(uVals, id)

	q := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = ?",
		table.Name,
		strings.Join(uSets, ", "),
		table.Pk,
	)

	res, err := e.db.Exec(q, uVals...)
	if err != nil {
		return nil, err
	}

	updated, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	return &PostTableRecordResponse{
		Updated: int(updated),
	}, nil
}

type DeleteTableRecordResponse struct {
	Deleted int `json:"deleted"`
}

func (e *DbExplorer) handleDeleteTableRecord(table Table, id int) (*DeleteTableRecordResponse, error) {
	q := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", table.Name, table.Pk)
	res, err := e.db.Exec(q, id)
	if err != nil {
		return nil, err
	}

	deleted, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	return &DeleteTableRecordResponse{
		Deleted: int(deleted),
	}, nil
}

func (e *DbExplorer) handleRequest(r *http.Request) (interface{}, error) {
	req, err := e.newRequest(r)
	if err != nil {
		return nil, err
	}

	switch r.Method {
	case http.MethodGet:
		if req.Table == nil {
			return e.handleGetTables()
		}

		if req.RecordId == nil {
			limit, offset := req.GetLimitOffset()
			return e.handleGetTableRecords(*req.Table, limit, offset)
		}

		return e.handleGetTableRecord(*req.Table, *req.RecordId)
	case http.MethodPut:
		if req.Table != nil {
			data, err := req.GetRecordData()
			if err != nil {
				return nil, err
			}

			return e.handlePutTableRecord(*req.Table, data)
		}
	case http.MethodPost:
		if req.Table != nil && req.RecordId != nil {
			data, err := req.GetRecordData()
			if err != nil {
				return nil, err
			}

			return e.handlePostTableRecord(*req.Table, *req.RecordId, data)
		}
	case http.MethodDelete:
		if req.Table != nil && req.RecordId != nil {
			return e.handleDeleteTableRecord(*req.Table, *req.RecordId)
		}
	}

	return nil, ResponseError{"method not found", 404}
}

func (e *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := Response{}

	data, err := e.handleRequest(r)
	if err == nil {
		res.Data = data
	} else {
		if re, ok := err.(ResponseError); ok {
			w.WriteHeader(re.StatusCode)
		}

		res.Error = err.Error()
	}

	jsonData, _ := json.Marshal(res)
	w.Write(jsonData)
}

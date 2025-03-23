package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

type User struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"company"`
	Country  string   `json:"country"`
	Email    string   `json:"email"`
	Job      string   `json:"job"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
}

func (u *User) UnmarshalJson(data []byte) error {
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	v := reflect.ValueOf(u).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonKey := field.Tag.Get("json")
		if jsonKey == "" {
			continue
		}

		if rawValue, exists := rawMap[jsonKey]; exists {
			fieldValue := v.Field(i)
			if !fieldValue.CanSet() {
				continue
			}

			switch fieldValue.Kind() {
			case reflect.String:
				if strVal, ok := rawValue.(string); ok {
					fieldValue.SetString(strVal)
				}
			case reflect.Slice:
				if slice, ok := rawValue.([]interface{}); ok {
					stringSlice := make([]string, len(slice))
					for i, v := range slice {
						if s, ok := v.(string); ok {
							stringSlice[i] = s
						}
					}
					fieldValue.Set(reflect.ValueOf(stringSlice))
				}
			}
		}
	}

	return nil
}

// вам надо написать более быструю оптимальную версию этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath) // OK
	if err != nil {
		panic(err)
	} // OK

	user := User{}
	scanner := bufio.NewScanner(file)
	browsers := make(map[string]bool)

	fmt.Fprintln(out, "found users:")

	i := 0
	// тратит много памяти, поэтому читаем построчно через сканнер
	for scanner.Scan() {
		if err := user.UnmarshalJson(scanner.Bytes()); err != nil {
			panic(err)
		}

		isAndroid := false
		isMSIE := false

		for _, browser := range user.Browsers {
			if strings.Contains(browser, "Android") {
				isAndroid = true
			} else if strings.Contains(browser, "MSIE") {
				isMSIE = true
			} else {
				continue
			}
			browsers[browser] = true
		}

		if isAndroid && isMSIE {
			email := strings.Replace(user.Email, "@", " [at] ", -1)
			fmt.Fprintf(out, fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email))
		}
		i++
	}

	fmt.Fprintln(out, "\nTotal unique browsers", len(browsers))
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
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

// вам надо написать более быструю оптимальную версию этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath) // OK
	if err != nil {
		panic(err)
	} // OK

	var i int = 0

	user := User{}
	scanner := bufio.NewScanner(file)
	browsers := make(map[string]bool)

	fmt.Fprintln(out, "found users:")

	// тратит много памяти, поэтому читаем построчно через сканнер
	for scanner.Scan() {
		// всё ещё долгий демаршалинг
		err := json.Unmarshal(scanner.Bytes(), &user)

		if err != nil {
			panic(err)
		}
		//if err := user.UnmarshalJson(scanner.Bytes()); err != nil {
		//	panic(err)
		//}
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

package main

import (
	"99_hw/pkg"
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath) // OK
	if err != nil {
		panic(err)
	} // OK

	user := pkg.User{}
	scanner := bufio.NewScanner(file)
	browsers := make(map[string]bool)

	fmt.Fprintln(out, "found users:")

	i := 0
	// тратит много памяти, поэтому читаем построчно через сканнер
	for scanner.Scan() {
		if err := user.UnmarshalJSON(scanner.Bytes()); err != nil {
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

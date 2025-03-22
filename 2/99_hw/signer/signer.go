package main

import (
	"fmt"
	"sort"
	"strings"
)

// job - анонимная функция, принимающая в качестве параметров входной и выходной каналы типа "пустой интерфейс"
// принимаем переменное количество job и выполняем их параллельно, изменяя в цикле значения входного и выходного каналов
func ExecutePipeline(jobs ...job) {
	var in, out chan interface{}
	// сначала последовательно запускаем все jobs
	for _, job := range jobs {
		in = out
		// не более 100 элементов в конвейере
		out = make(chan interface{}, 100)
		// запускаем текущую job
		job(in, out)
		close(out)
	}
}

// параллельно читаем все строки из входного потока
func ProcessString(in, out chan interface{}, str func(string string) string) {
	// проходимся по всем значениям из входного канала
	for data := range in {
		// форматируем значение в строку, если это возможно(Sprintf)
		formattedData := fmt.Sprintf("%v", data)
		// выводим значение в выходной канал
		out <- str(formattedData)
	}
}

func SingleHash(in, out chan interface{}) {
	// выполняем анонимную функцию , возвращающую строку
	ProcessString(in, out, func(data string) string {
		crc32 := DataSignerCrc32(data)
		md5 := DataSignerCrc32(DataSignerMd5(data))
		return fmt.Sprintf("%s~%s", crc32, md5)
	})
}

func MultiHash(in, out chan interface{}) {
	// выполняем анонимную функцию , возвращающую строку
	ProcessString(in, out, func(data string) string {
		var sb strings.Builder
		for th := 0; th <= 5; th++ {
			formatted := fmt.Sprintf("%v%s", th, data)
			crc32 := DataSignerCrc32(formatted)
			sb.WriteString(crc32)
		}

		return sb.String()
	})
}

// формируем ответ из всех значений входного канала и выводим в выходной
func CombineResults(in, out chan interface{}) {
	var ans []string
	for data := range in {
		formattedData := fmt.Sprintf("%v", data)
		ans = append(ans, formattedData)
	}
	sort.Strings(ans)
	out <- strings.Join(ans, "_")
}

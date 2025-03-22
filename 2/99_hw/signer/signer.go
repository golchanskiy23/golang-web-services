package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// job - анонимная функция, принимающая в качестве параметров входной и выходной каналы типа "пустой интерфейс"
// принимаем переменное количество job и выполняем их параллельно, изменяя в цикле значения входного и выходного каналов
func ExecutePipeline(jobs ...job) {
	var in, out chan interface{}
	wg := &sync.WaitGroup{}
	// сначала последовательно запускаем все jobs
	for _, job_ := range jobs {
		in = out
		// не более 100 элементов в конвейере
		out = make(chan interface{}, 100)
		wg.Add(1)
		// запускаем текущую job, оборачивая её в горутину
		// не закрываем канал - range на следующей иерации будет бесконечно ждать ввода из потока
		go func(j job, in, out chan interface{}) {
			defer wg.Done()
			defer close(out)
			j(in, out)
		}(job_, in, out)
	}
	wg.Wait()
}

// параллельно читаем все строки из входного потока
func ProcessString(in, out chan interface{}, str func(string string) string) {
	wg := &sync.WaitGroup{}
	// проходимся по всем значениям из входного канала
	for data := range in {
		wg.Add(1)
		// форматируем значение в строку, если это возможно(Sprintf)
		go func(formattedData string) {
			defer wg.Done()
			// выводим значение в выходной канал
			out <- str(formattedData)
		}(fmt.Sprintf("%v", data))
	}
	wg.Wait()
}

// можно передавать вычисленные значения хэшей в небуферизированные каналы, вызывая блокировку при чтении из пустого
func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	mtx := &sync.Mutex{}
	// выполняем анонимную функцию , возвращающую строку
	ProcessString(in, out, func(data string) string {
		var crc32, md5 string

		wg.Add(2)
		go func() {
			defer wg.Done()
			crc32 = DataSignerCrc32(data)
		}()

		go func() {
			defer wg.Done()
			var md5TMP string

			mtx.Lock()
			md5TMP = DataSignerMd5(data)
			mtx.Unlock()
			md5 = DataSignerCrc32(md5TMP)
		}()
		wg.Wait()
		return fmt.Sprintf("%s~%s", crc32, md5)
	})
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	// выполняем анонимную функцию , возвращающую строку
	ProcessString(in, out, func(data string) string {
		ans := make([]string, 6)
		for th := 0; th <= 5; th++ {
			wg.Add(1)
			go func(th int) {
				defer wg.Done()
				formatted := DataSignerCrc32(fmt.Sprintf("%v%s", th, data))
				ans[th] = formatted
			}(th)
		}
		wg.Wait()
		return strings.Join(ans, "")
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

package util

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"

	"golang.org/x/exp/constraints"
)

type Pair[T constraints.Ordered] struct {
	Key   string
	Value T
}

type PairList[T constraints.Ordered] []Pair[T]

func (p PairList[T]) Len() int               { return len(p) }
func (p PairList[T]) Less(i int, j int) bool { return p[i].Value < p[j].Value }
func (p PairList[T]) Swap(i int, j int)      { p[i], p[j] = p[j], p[i] }

func Cleanup(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		log.Printf("Error while cleaning up: %v", err)
	}
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

type fileNameAndData struct {
	fileName string
	data     string
}

// MultipleFileRead reads multiple files concurrently
// and returns a map of file names to file contents
func MultipleFileRead(fileNames []string) (map[string]string, error) {
	var wg sync.WaitGroup

	numberOfFiles := len(fileNames)

	dataChannel := make(chan fileNameAndData, numberOfFiles)
	errChannel := make(chan error, numberOfFiles)

	readFile := func(filename string) {
		defer wg.Done()
		data, err := os.ReadFile(filename)
		if err != nil {
			errChannel <- err
			return
		}
		out := fileNameAndData{
			fileName: filename,
			data:     string(data),
		}
		dataChannel <- out
	}

	wg.Add(numberOfFiles)

	for _, filename := range fileNames {
		go readFile(filename)
	}

	wg.Wait()

	close(dataChannel)
	close(errChannel)

	var errStrings []string
	for err := range errChannel {
		if err != nil {
			errStrings = append(errStrings, err.Error())
		}
	}
	// Process errors
	if len(errStrings) > 0 {
		return nil, fmt.Errorf("errors occurred: %v", errStrings)
	}

	// Process data
	allData := make(map[string]string)
	for data := range dataChannel {
		allData[data.fileName] = data.data
	}
	return allData, nil
}

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func Filter[T any](ts []T, f func(T) bool) []T {
	var us []T
	for _, t := range ts {
		if f(t) {
			us = append(us, t)
		}
	}
	return us
}

func RandomDrawWithoutReplacement[T any](arr []T, n int) []T {
	if n > len(arr) {
		n = len(arr)
	}

	// Create a copy of the original array
	arrCopy := make([]T, len(arr))
	copy(arrCopy, arr)

	// Shuffle the array copy
	rand.Shuffle(len(arrCopy), func(i, j int) { arrCopy[i], arrCopy[j] = arrCopy[j], arrCopy[i] })

	// Take the first n items from the shuffled array
	return arrCopy[:n]
}

func Contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T constraints.Ordered](a, b T) T {
	if a < b {
		return b
	}
	return a
}

package gotail

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/bmizerany/assert"
)

var fname string = "test.log"
var wg sync.WaitGroup

func TestAppendFile(t *testing.T) {
	createFile("")
	defer removeFile()

	tail, err := NewTail(fname, Config{Timeout: 10})
	assert.Equal(t, err, nil)

	var line string

	done := make(chan bool)

	go func() {
		line = <-tail.Lines
		done <- true
		return
	}()

	writeFile("foobar\n")

	<-done

	assert.Equal(t, "foobar", line)

}

func TestWriteNewFile(t *testing.T) {
	var tail *Tail
	var line string
	done := make(chan bool)

	go func() {
		tail, _ = NewTail(fname, Config{Timeout: 10})

		line = <-tail.Lines
		done <- true
		return
	}()

	time.Sleep(10 * time.Millisecond) // Allow the listener to fully setup
	createFile("")
	defer removeFile()

	writeFile("foobar\n")

	<-done

	assert.Equal(t, "foobar", line)
}

func TestRenameFile(t *testing.T) {
	var tail *Tail
	var line string
	done := make(chan bool)

	// Sets up background tailer
	go func() {
		tail, _ = NewTail(fname, Config{Timeout: 10})

		line = <-tail.Lines
		done <- true
		return
	}()

	createFile("")
	renameFile()

	_, err := os.Stat(fname)
	assert.Equal(t, true, os.IsNotExist(err))

	time.Sleep(10 * time.Millisecond) // Allow the listener to fully setup
	createFile("foobar\n")

	<-done

	assert.Equal(t, "foobar", line)

	_ = os.Remove(fname + "_new")
	removeFile()
}

func TestNoFile(t *testing.T) {
	_, err := NewTail(fname, Config{Timeout: 0})
	assert.Equal(t, true, os.IsNotExist(err))
}

func TestBenchmark(t *testing.T) {
	log.Println("Running Benchmarks")
	var concurrency int = 2
	var rowcount = 500000           // number of rows to write
	runtime.GOMAXPROCS(concurrency) // number of writer processes
	var row int

	_ = os.Remove(fname)
	createFile("")

	f, err := os.OpenFile(fname, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalln(err)
	}

	tail, err := NewTail(fname, Config{Timeout: 10})
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < concurrency; i++ {
		go func(i int) {
			log.Printf("Spawned %d Workers\n", i)

			for row <= rowcount {
				body := fmt.Sprintf("%d Worker doing the write.  %d iteration is being written.\n", i, row)
				writeContents(f, body)
				row++
			}
		}(i)
	}

	count := 0
	startTime := time.Now()

	for _ = range tail.Lines {
		count++
		if count == rowcount {
			break
		}
	}

	duration := time.Since(startTime).Seconds()
	fmt.Printf("%d rows processed in %0.4f seconds, at the rate of %0.4f rows/s\n", count, duration, float64(count)/duration)
	removeFile()
}

func writeContents(f *os.File, contents string) {
	_, err := f.WriteString(contents)
	if err != nil {
		log.Fatalln(err)
	}
}

func createFile(contents string) {
	err := ioutil.WriteFile(fname, []byte(contents), 0600)
	if err != nil {
		log.Fatalln(err)
	}
}

func removeFile() {
	err := os.Remove(fname)
	if err != nil {
		log.Fatalln(err)
	}
}

func renameFile() {
	oldname := fname
	newname := fname + "_new"
	err := os.Rename(oldname, newname)
	if err != nil {
		log.Fatalln(err)
	}
}

func writeFile(contents string) {
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	writeContents(f, contents)
}

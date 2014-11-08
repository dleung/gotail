package gotail

import (
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bmizerany/assert"
)

var fname string = "test.log"
var wg sync.WaitGroup

func TestAppendFile(t *testing.T) {
	CreateFile("")
	defer RemoveFile()

	tail, err := NewTail(fname, Config{timeout: 10})
	assert.Equal(t, err, nil)

	var line string

	done := make(chan bool)

	go func() {
		line = <-tail.Lines
		done <- true
		return
	}()

	WriteFile("foobar\n")

	<-done

	assert.Equal(t, "foobar", line)

}

func TestWriteNewFile(t *testing.T) {
	var tail *Tail
	var line string
	done := make(chan bool)

	go func() {
		tail, _ = NewTail(fname, Config{timeout: 10})

		line = <-tail.Lines
		done <- true
		return
	}()

	time.Sleep(10 * time.Millisecond) // Allow the listener to fully setup
	CreateFile("")
	defer RemoveFile()

	WriteFile("foobar\n")

	<-done

	assert.Equal(t, "foobar", line)
}

func TestRenameFile(t *testing.T) {
	var tail *Tail
	var line string
	done := make(chan bool)

	// Sets up background tailer
	go func() {
		tail, _ = NewTail(fname, Config{timeout: 10})

		line = <-tail.Lines
		done <- true
		return
	}()

	CreateFile("")
	RenameFile()

	_, err := os.Stat(fname)
	assert.Equal(t, true, os.IsNotExist(err))

	time.Sleep(10 * time.Millisecond) // Allow the listener to fully setup
	CreateFile("foobar\n")

	<-done

	assert.Equal(t, "foobar", line)

	_ = os.Remove(fname + "_new")
	RemoveFile()
}

func TestNewTail(t *testing.T) {
	_, err := NewTail(fname, Config{timeout: 0})
	assert.Equal(t, true, os.IsNotExist(err))
}

func CreateFile(contents string) {
	err := ioutil.WriteFile(fname, []byte(contents), 0600)
	if err != nil {
		log.Fatalln(err)
	}
}

func RemoveFile() {
	err := os.Remove(fname)
	if err != nil {
		log.Fatalln(err)
	}
}

func RenameFile() {
	oldname := fname
	newname := fname + "_new"
	err := os.Rename(oldname, newname)
	if err != nil {
		log.Fatalln(err)
	}
}

func WriteFile(contents string) {
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	_, err = f.WriteString(contents)
	if err != nil {
		log.Fatalln(err)
	}
}

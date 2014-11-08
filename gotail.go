package gotail

import (
	"bufio"
	"strings"

	"io"
	"log"
	"os"
	"time"

	"code.google.com/p/go.exp/fsnotify"
)

type Tail struct {
	Lines chan string

	reader  *bufio.Reader
	watcher *fsnotify.Watcher
	fname   string
	file    *os.File
	config  Config
}

type Config struct {
	Timeout int
}

// Creates a new Tail Object which starts tailing the file
func NewTail(fname string, config Config) (*Tail, error) {
	tail := &Tail{
		Lines:  make(chan string),
		fname:  fname,
		config: config,
	}

	err := tail.openAndWatch()

	if err != nil {
		return nil, err
	}

	tail.listenAndReadLines()

	return tail, nil
}

// Close the tail object when finished, closing the file handle and watcher
func (t *Tail) Close() {
	t.file.Close()
	t.watcher.Close()
}

// Opens and watch the file with timeout.  If the timeout is reached, it exits with the error
func (t *Tail) openAndWatch() error {
	var err error
	var newFile bool

	timeout := make(chan error, 1)

	go func() {
		for {
			err = t.openFile(newFile)
			if err != nil {
				if os.IsNotExist(err) && newFile == false {
					newFile = true
				}

				if t.config.Timeout == 0 {
					timeout <- err
					return
				} else {
					continue
				}

			}
			newFile = false

			err = t.watchFile()

			if err == nil {
				timeout <- nil
				return
			}
		}
	}()

	if t.config.Timeout != 0 {
		go func() {
			time.Sleep(time.Duration(t.config.Timeout) * time.Second)

			timeout <- err
		}()
	}

	select {
	case err := <-timeout:
		if err != nil {
			return err
		}
	}

	return nil
}

// Opens a file and finds the last byte to start tailing
func (t *Tail) openFile(newFile bool) (err error) {
	if t.file != nil {
		t.file.Close()
	}

	t.file, err = os.Open(t.fname)
	if err != nil {
		return err
	}

	// If it's a new file, then start reading the file from the beginning.
	// This is because sometimes, a new file is considered "MODIFY" and
	// file.Seek will automatically point to the last byte of the file,
	// Skipping the first line
	if !newFile {
		_, err = t.file.Seek(0, 2)
	}

	if err != nil {
		return err
	}

	t.reader = bufio.NewReader(t.file)

	return nil
}

// Assigns a new watcher to the file
func (t *Tail) watchFile() error {
	if t.watcher != nil {
		t.watcher.Close()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	t.watcher = watcher
	err = t.watcher.Watch(t.fname)

	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case evt := <-t.watcher.Event:
				if evt != nil && (evt.IsDelete() || evt.IsRename()) {
					if err = t.openAndWatch(); err != nil {
						log.Fatalln("open and watch failed:", err)
					}
				}
			case err := <-t.watcher.Error:
				if err != nil {
					log.Fatalln("Watcher err:", err)
				}
			}
		}
	}()

	return nil
}

// Separate goroutine to listen for new lines
func (t *Tail) listenAndReadLines() {
	go func() {
		for {
			if t.reader == nil {
				continue
			}

			line, err := t.reader.ReadString('\n')

			if err == io.EOF {
				continue
			}

			t.Lines <- strings.TrimRight(line, "\n")
		}
	}()
}

package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type copyResult struct {
	file   string
	result bool
	bytes  int64
	target string
}

type resultList struct {
	Items []copyResult
}

type fileResult struct {
	file string
}

func md5sum(filePath string) string {
	input, _ := os.Open(filePath)
	hash := sha256.New()
	defer input.Close()
	if _, err := io.Copy(hash, input); err != nil {
		log.Fatal(err)
	}
	sum := hash.Sum(nil)

	fmt.Println("md5sum", filePath, getGID(), sum)
	return filePath
}

func BatchFiles(values <-chan string, maxItems int, maxTimeout time.Duration) chan []string {
	batches := make(chan []string)

	go func() {
		defer close(batches)

		for keepGoing := true; keepGoing; {
			var batch []string
			expire := time.After(maxTimeout)
			for {
				select {
				case value, ok := <-values:
					if !ok {
						fmt.Println("all done")
						keepGoing = false
						goto done
					}

					batch = append(batch, value)
					if len(batch) == maxItems {
						fmt.Println("reached batch")
						goto done
					}

				case <-expire:
					fmt.Println("expired")
					keepGoing = false
					goto done
				}
			}

		done:
			if len(batch) > 0 {
				fmt.Println("done")
				batches <- batch
			}
		}
	}()
	fmt.Println("returning batch")
	return batches
}

func walker(Dir string, maxJobs int) <-chan string {
	out := make(chan string, 1)
	log.Println("walker")
	go func() {
		err := filepath.Walk(Dir,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					log.Println(err)
					return err
				}
				if info.IsDir() == true {
					return nil
				}
				out <- path
				return nil
			})
		if err != nil {
			close(out)
			log.Println(err)
		}
		close(out)
	}()
	return out
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func process(filebatch []string) {
	var wg sync.WaitGroup
	for _, dirname := range filebatch {
		wg.Add(1)
		go func() {
			defer wg.Done()
			md5sum(dirname)
		}()
	}
	wg.Wait()
}

func main() {
	var (
		maxJobs int
		Dir     string
	)
	cores := runtime.GOMAXPROCS(0)
	flag.StringVar(&Dir, "dir", "/tmp/testdir", "Directory")
	flag.IntVar(&maxJobs, "jobs", cores, "Directory")
	flag.Parse()

	processed := walker(Dir, maxJobs)
	for batch := range BatchFiles(processed, maxJobs, 20*time.Millisecond) {
		fmt.Println(batch)
		process(batch)
	}
}

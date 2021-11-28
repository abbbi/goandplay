package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

func process(filebatch []string, maxJobs int) {
	limChan := make(chan bool, maxJobs)
	for i := 0; i < maxJobs; i++ {
		limChan <- true
	}
	for _, dirname := range filebatch {
		<-limChan
		go md5sum(dirname)
		limChan <- true
	}
}

func main() {
	var (
		maxJobs int
		Dir     string
	)
	flag.StringVar(&Dir, "dir", "/tmp/testdir", "Directory")
	flag.IntVar(&maxJobs, "jobs", 5, "Directory")
	flag.Parse()

	processed := walker(Dir, maxJobs)
	for batch := range BatchFiles(processed, maxJobs, 20*time.Millisecond) {
		fmt.Println(batch)
		process(batch, maxJobs)
	}
}

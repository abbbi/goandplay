package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type sumResult struct {
	file string
	sum  string
}

func sha256sum(filePath string) sumResult {
	input, _ := os.Open(filePath)
	hash := sha256.New()
	defer input.Close()
	if _, err := io.Copy(hash, input); err != nil {
		log.Fatal(filePath, err)
	}
	return sumResult{
		file: filePath,
		sum:  fmt.Sprintf("%x", hash.Sum(nil)),
	}
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
						keepGoing = false
						goto done
					}
					batch = append(batch, value)
					if len(batch) == maxItems {
						goto done
					}
				case <-expire:
					keepGoing = false
					goto done
				}
			}
		done:
			if len(batch) > 0 {
				batches <- batch
			}
		}
	}()
	return batches
}

func walker(Dir string, maxJobs int) <-chan string {
	out := make(chan string, 1)
	go func() {
		err := filepath.Walk(Dir,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					log.Println(err)
					return err
				}
				if !info.Mode().IsRegular() {
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

func process(filebatch []string) chan sumResult {
	out := make(chan sumResult, len(filebatch))
	var wg sync.WaitGroup
	defer close(out)
	for _, dirname := range filebatch {
		wg.Add(1)
		go func(dirname string) {
			defer wg.Done()
			out <- sha256sum(dirname)
		}(dirname)
	}
	wg.Wait()
	return out
}

func main() {
	var (
		maxJobs int
		Dir     string
	)
	cores := runtime.GOMAXPROCS(0)
	flag.StringVar(&Dir, "dir", "/tmp/", "Directory")
	flag.IntVar(&maxJobs, "jobs", cores, "Directory")
	flag.Parse()

	fmt.Println("Cores:", cores)
	processed := walker(Dir, maxJobs)
	for batch := range BatchFiles(processed, maxJobs, 20*time.Millisecond) {
		for result := range process(batch) {
			fmt.Printf("Sum: [%s] [%s]\n", result.file, result.sum)
		}
	}
}

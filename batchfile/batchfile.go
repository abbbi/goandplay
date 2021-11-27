package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
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

func md5sum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func worker(concurrency int, filename <-chan string) <-chan string {
	log.Println("started worker")
	var wg sync.WaitGroup
	wg.Add(1)
	out := make(chan string, concurrency)
	go func() {
		limiter := make(chan bool, concurrency)
		for i := 0; i < concurrency; i++ {
			limiter <- true
		}

		for i := 0; i < concurrency; i++ {
			<-limiter
			path := <-filename
			log.Println(path)
			go md5sum(path)
			limiter <- true
		}
		wg.Done()
	}()
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func walker(Dir string, maxJobs int) <-chan string {
	out := make(chan string, maxJobs)
	log.Println("walker")

	go func() {
		err := filepath.Walk(Dir,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				log.Println(path)
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

func main() {
	var (
		maxJobs int
		Dir     string
	)

	flag.StringVar(&Dir, "dir", "/tmp/testdir", "Directory")
	flag.IntVar(&maxJobs, "jobs", 5, "Directory")

	flag.Parse()

	walker := walker(Dir, maxJobs)
	processed := worker(maxJobs, walker)

	for range processed {
	}

}

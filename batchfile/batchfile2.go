package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

func worker(concurrency int, in <-chan []string) <-chan string {
	log.Println("started worker")
	var wg sync.WaitGroup
	wg.Add(1)
	out := make(chan string)
	work := func(sbatch []string) {
		log.Println("Process", len(sbatch), "files")
		for _, fileName := range sbatch {
			log.Println("Working on:", fileName)
			input, err := os.Open(fileName)
			if err == nil {
				foo, _ := ioutil.TempFile("", "abifile")
				dst, err := os.Create(foo.Name())
				defer dst.Close()
				if err == nil {
					bytes, err := io.Copy(dst, input)
					if err != nil {
						fmt.Println(err)
					}
					log.Println("done with:", fileName, bytes, foo.Name())
				}
			} else {
				fmt.Println(err)
			}
			out <- fileName
		}
		wg.Done()
	}
	go func() {
		for data := range in {
			sbatch := data
			log.Println("starting work")
			go work(sbatch)
		}
	}()
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func walker() <-chan []string {
	out := make(chan []string)

	files := []string{}
	go func() {
		err := filepath.Walk("/tmp/testdir",
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				files = append(files, path)
				if len(files) >= int(5) {
					retFiles := files
					fmt.Println("sending batch:", retFiles)
					files = nil
					out <- retFiles
				}
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
		maxJobs int = 5
		Dir     string
	)

	flag.StringVar(&Dir, "dir", "/tmp/testdir", "Directory")

	flag.Parse()

	files, err := ioutil.ReadDir(Dir)
	if err != nil {
		log.Fatal(err)
	}
	if len(files) < maxJobs {
		maxJobs = len(files)
	}

	walker := walker()
	processed := worker(maxJobs, walker)

	for file := range processed {
		log.Println(file)
	}
}

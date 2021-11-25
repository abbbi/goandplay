package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
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

func worker(id string, results chan<- copyResult) {
	source, err := os.Open(id)
	if err != nil {
		results <- copyResult{id, false, 0, ""}
		return
	}
	defer source.Close()

	tmpfile, _ := ioutil.TempFile("", "abitest")
	destination, err := os.Create(tmpfile.Name())
	if err != nil {
		results <- copyResult{id, false, 0, ""}
		return
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	if err != nil {
		results <- copyResult{id, false, 0, ""}
		return
	}
	results <- copyResult{id, true, nBytes, tmpfile.Name()}
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

	allResults := resultList{}
	for i := 0; i < len(files); i += maxJobs {
		j := i + maxJobs
		if j > len(files) {
			j = len(files)
		}
		batch := files[i:j]
		log.Println("Starting", len(batch), "jobs")

		results := make(chan copyResult, len(batch))

		for _, file := range batch {
			fp := fmt.Sprintf("%s/%s", Dir, file.Name())
			go worker(fp, results)
		}

		for range batch {
			allResults.Items = append(allResults.Items, <-results)
		}
	}

	log.Printf("Processed %d files:", len(allResults.Items))
	for _, f := range allResults.Items {
		if f.result == true {
			log.Printf("OK: %s -> %s (%d)", f.file, f.target, f.bytes)
		} else {
			log.Printf("NOK: %s", f.file)
		}
	}
}

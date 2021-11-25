package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

func worker(id string, results chan<- string) {
	log.Println("file:", id, "started  job")

	source, err := os.Open(id)
	if err != nil {
		log.Printf("error")
	}
	defer source.Close()

	tmpfile, _ := ioutil.TempFile("", "abitest")
	destination, err := os.Create(tmpfile.Name())
	if err != nil {
		log.Println("err")
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	if err != nil {
		log.Println("err")
	}
	log.Println(nBytes)

	log.Println("file", id, "finished job")
	results <- id
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

	for i := 0; i < len(files); i += maxJobs {
		j := i + maxJobs
		if j > len(files) {
			j = len(files)
		}
		batch := files[i:j]
		log.Println("Starting", len(batch), "jobs")

		results := make(chan string, len(batch))

		for _, file := range batch {
			fp := fmt.Sprintf("%s/%s", Dir, file.Name())
			go worker(fp, results)
		}

		for range batch {
			foo := <-results
			log.Printf("Done: %s", foo)
		}
	}
}

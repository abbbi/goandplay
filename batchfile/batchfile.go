package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func worker(id string, results chan<- string) {
	fmt.Println("file:", id, "started  job")

	source, err := os.Open(id)
	if err != nil {
		fmt.Printf("error")
	}
	defer source.Close()

	tmpfile, _ := ioutil.TempFile("", "abitest")
	destination, err := os.Create(tmpfile.Name())
	if err != nil {
		fmt.Println("err")
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	if err != nil {
		fmt.Println("err")
	}
	fmt.Println(nBytes)

	fmt.Println("file", id, "finished job")
	results <- id
}

func main() {
	var (
		maxJobs int = 5
	)
	files := [...]string{
		"testfile1",
		"testfile2",
		"testfile3",
		"testfile4",
		"testfile5",
		"testfile6",
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
		fmt.Println("Batch:", batch)
		fmt.Println("Starting", len(batch), "jobs")

		jobs := make(chan string, len(batch))
		results := make(chan string, len(batch))

		for _, file := range batch {
			go worker(file, results)
		}

		for _, file := range batch {
			jobs <- file
		}
		close(jobs)

		for range batch {
			<-results
		}

	}
}

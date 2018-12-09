package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	numWorker = 3
)

func main() {

	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	readers := make([]io.Reader, numWorker)
	writers := make([]io.WriteCloser, numWorker)

	for i := 0; i < numWorker; i++ {
		pr, pw := io.Pipe()
		readers[i] = pr
		writers[i] = pw
	}

	for i := 0; i < numWorker; i++ {
		go postBin(i, readers[i])
	}

	go func() {
		defer func() {
			for _, w := range writers {
				w.Close()
			}
		}()

		resp, err := http.DefaultClient.Get("https://api.data.gov.sg/v1/environment/air-temperature")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		iowriters := make([]io.Writer, len(writers))
		for i, w := range writers {
			iowriters[i] = w.(io.Writer)
		}

		written, err := io.Copy(io.MultiWriter(iowriters...), resp.Body)
		fmt.Printf("written %d bytes to multiwriter...\n", written)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	<-sigterm
	fmt.Println("exiting...")
}

func postBin(workerNo int, reader io.Reader) {

	fmt.Printf("worker %d starting ...\n", workerNo)
	req, err := http.NewRequest("POST", "http://localhost:3030/post", reader)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("worker %d prepared request ...\n", workerNo)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("worker %d sent request ...\n", workerNo)

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("worker %d read body ...\n", workerNo)

	fmt.Printf("worker %d received http response status code %d, bytes: %d\n", workerNo, resp.StatusCode, len(bytes))
}

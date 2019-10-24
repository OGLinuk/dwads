package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	dwspb "./proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

var (
	dwsClient dwspb.DWSServiceClient
)

func UploadWADHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Processing uwad ...")
	f, fh, err := r.FormFile("file")
	if err != nil {
		log.Printf("Failed to get r.FormFile: %s", err.Error())
	}

	stream, err := dwsClient.UploadWAD(context.Background())
	if err != nil {
		log.Printf("Failed to dwsClient.Upload: %s", err.Error())
	}

	err = stream.Send(&dwspb.UploadWADRequest{
		Data: []byte(fh.Filename),
	})

	if err != nil {
		err = errors.Wrapf(err,
			"failed to send fh.Filename via stream")

		return
	}

	buf := make([]byte, 128)

	for {
		n, err := f.Read(buf)
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}

			err = errors.Wrapf(err,
				"errored while copying from file to buf")

			log.Println(err)
		}

		err = stream.Send(&dwspb.UploadWADRequest{
			Data: buf[:n],
		})
		if err != nil {
			err = errors.Wrapf(err,
				"failed to send chunk via stream")

			log.Println(err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		err = errors.Wrapf(err,
			"failed to receive upstream status response")
		log.Println(err)
	}

	if resp.Success != true {
		err = errors.Wrapf(err,
			"success was not true")
		log.Println(err)
	}

	log.Println("Upload was successful ...")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	port := flag.Int("p", 9042, "port to serve on")
	godaemon := flag.Bool("d", false, "run as a daemon")
	directory := flag.String("dir", "wads", "the directory of static file to host")

	flag.Parse()

	conn, err := grpc.Dial("0.0.0.0:9001", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dial grpc server: %v", err)
	}

	dwsClient = dwspb.NewDWSServiceClient(conn)

	if *godaemon {
		args := os.Args[1:]
		for i := 0; i < len(args); i++ {
			if args[i] == "-d" {
				args = append(args[:i], args[i+1:]...)
				break
			}
		}
		cmd := exec.Command(os.Args[0], args...)
		cmd.Start()
		fmt.Printf("Server running...\nClose server: kill -9 %d\n", cmd.Process.Pid)
		fmt.Printf("http://localhost:%d \n", *port)
		*godaemon = false
		os.Exit(0)
	}

	http.Handle("/", http.FileServer(http.Dir(*directory)))
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/upload.html")
	})
	http.HandleFunc("/uwad", UploadWADHandler)

	log.Printf("Starting (D)OOM (WAD) (S)tore on port: %d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil), nil)
}

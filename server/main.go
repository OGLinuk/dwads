package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

	dwspb "../proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type server struct{}

func (s *server) UploadWAD(stream dwspb.DWSService_UploadWADServer) error {
	log.Println("UploadWAD (gRPC server) processing ...")
	fname, err := stream.Recv()
	if err != nil {
		err = errors.Wrapf(err,
			"failed while reading fname from stream")
		return err
	}

	f, err := os.Create(fmt.Sprintf("wads/%s", string(fname.GetData())))
	if err != nil {
		err = errors.Wrapf(err,
			"failed to os.Create")
		return err
	}

	for {
		recvd, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			err = errors.Wrapf(err,
				"failed while reading chunks from stream")
			return err
		}

		f.Write(recvd.GetData())
	}

	err = stream.SendAndClose(&dwspb.UploadWADResponse{
		Success: true,
	})
	if err != nil {
		err = errors.Wrapf(err,
			"failed to send status code")
		return err
	}

	return nil
}

func main() {
	lis, err := net.Listen("tcp", "0.0.0.0:9001")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("Server running on port 9001 ...")

	srvr := grpc.NewServer()
	dwspb.RegisterDWSServiceServer(srvr, &server{})
	if err := srvr.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

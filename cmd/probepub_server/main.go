package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	pb "github.com/europelee/grpcplay/internal/probepub"
	"google.golang.org/grpc"
)

var (
	logRecord = flag.Bool("record_logging", true, "logging probe-record, else not")
	port      = flag.Int("port", 10000, "The server port")
)

type server struct{}

func (s *server) PublishRTT(stream pb.ProbePub_PublishRTTServer) error {
	var count int32
	startTime := time.Now()
	for {
		r, err := stream.Recv()
		if err == io.EOF {
			endTime := time.Now()
			return stream.SendAndClose(
				&pb.PubStat{
					ProbeCount:  count,
					ElapsedTime: int32(endTime.Sub(startTime).Seconds()),
				},
			)
		}
		if err != nil {
			return err
		}
		if *logRecord {
			log.Println(r)
		}
		count++
	}
}
func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterProbePubServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

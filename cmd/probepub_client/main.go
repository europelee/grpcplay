package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"time"

	pb "github.com/europelee/grpcplay/internal/probepub"
	"google.golang.org/grpc"
)

var (
	serverAddr = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")
)

func runPubRtt(client pb.ProbePubClient) {
	// Create a random number of random points
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	count := int(r.Int31n(2)) + 2 // Traverse at least two points
	var recrods []*pb.RTTRecord
	for i := 0; i < count; i++ {
		recrods = append(recrods,
			&pb.RTTRecord{
				Channel: "mix_ping",
				Ts:      time.Now().Unix(),
				Vip:     "1.1.1.1",
				Qip:     "3.3.3.3",
				Method:  1,
				Rtt:     int32(i),
				Hop:     int32(i) + 1,
			},
		)
	}
	log.Printf("publish %d recrods.", len(recrods))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.PublishRTT(ctx)
	if err != nil {
		log.Fatalf("%v.PublishRTT(_) = _, %v", client, err)
	}
	for _, r := range recrods {
		if err := stream.Send(r); err != nil {
			log.Fatalf("%v.Send(%v) = %v", stream, r, err)
		}
	}
	reply, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
	}
	log.Printf("pub summary: %v", reply)
}

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewProbePubClient(conn)
	t := time.NewTicker(time.Duration(5) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			runPubRtt(c)
		}
	}
}

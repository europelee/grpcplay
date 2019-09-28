package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime/debug"
	"time"

	"google.golang.org/grpc/connectivity"

	pb "github.com/europelee/grpcplay/internal/probepub"
	"google.golang.org/grpc"
)

var (
	loopNum    = flag.Int("loop_num", 3, "loop num")
	interval   = flag.Int("pub_interval", 5, "publish interval(seconds)")
	minRecNum  = flag.Int("min_record_num", 1000, "min record number each collect time")
	serverAddr = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")
)

func runPubRtt(s pb.ProbePub_PublishRTTClient) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf(fmt.Sprintf("exception: %s\n", r))
			log.Printf(fmt.Sprintf("%s", debug.Stack()))
			os.Exit(-1)
		}
	}()
	if s == nil {
		return fmt.Errorf("s==nil")
	}
	// Create a random number of random points
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	count := int(r.Int31n(500)) + *minRecNum // Traverse at least two points
	var recrods []*pb.RTTRecord
	for i := 0; i < count; i++ {
		chn := fmt.Sprintf("mix_ping|%d", os.Getpid())
		recrods = append(recrods,
			&pb.RTTRecord{
				Channel: chn,
				Ts:      time.Now().Unix(),
				Vip:     "1.1.1.1",
				Qip:     "3.3.3.3",
				Method:  1,
				Rtt:     int32(i),
				Hop:     int32(i) + 1,
			},
		)
	}

	for _, r := range recrods {
		if err := s.Send(r); err != nil {
			log.Printf("%v.Send(%v) = %v", s, r, err)
			return err
		}
	}
	log.Printf("finish publishing %d recrods.", len(recrods))
	return nil
}

func cleanNetStream(curConn *grpc.ClientConn, curStream pb.ProbePub_PublishRTTClient) {
	if curConn != nil {
		log.Printf("curConn.GetState()==%s", curConn.GetState().String())
		if curConn.GetState() != connectivity.Shutdown {
			if err := curConn.Close(); err != nil {
				log.Printf("close conn fail %v", err.Error())
			}
		}
	}
	if curStream != nil {
		if _, err := curStream.CloseAndRecv(); err != nil {
			log.Printf("close stream fail %v", err.Error())
		}
	}
}
func createNetStream() (*grpc.ClientConn, pb.ProbePub_PublishRTTClient, error) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(*serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return nil, nil, err
	}
	c := pb.NewProbePubClient(conn)
	stream, err := c.PublishRTT(context.Background())
	if err != nil {
		log.Printf("%v.PublishRTT(_) = _, %v", c, err)
		conn.Close()
		return nil, nil, err
	}
	return conn, stream, nil
}

func main() {
	flag.Parse()
	conn, stream, _ := createNetStream()
	t := time.NewTicker(time.Duration(*interval) * time.Second)
	defer t.Stop()
	loopIdx := 0
LOOP:
	for {
		select {
		case <-t.C:
			err := runPubRtt(stream)
			if err != nil {
				cleanNetStream(conn, stream)
				conn, stream, _ = createNetStream()
			}
			if *loopNum > 0 {
				loopIdx++
				if loopIdx >= *loopNum {
					log.Print("end")
					break LOOP
				}
			}

		}
	}
	if stream != nil {
		reply, err := stream.CloseAndRecv()
		if err != nil {
			log.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
		}
		log.Printf("pub summary: %v", reply)
	}
	if conn != nil {
		conn.Close()
	}

}

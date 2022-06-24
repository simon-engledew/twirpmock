package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/simon-engledew/twirpmock/pkg/handler"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

func unmarshalSet(filename string, set *descriptorpb.FileDescriptorSet) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	return proto.Unmarshal(data, set)
}

func run(setPath, scriptPath, host string, port int) error {
	var set descriptorpb.FileDescriptorSet

	if err := unmarshalSet(setPath, &set); err != nil {
		return err
	}

	mux := handler.NewServeMux()

	if err := mux.Handle(&set, scriptPath, nil); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	ctx := context.Background()

	server := &http.Server{
		Handler:      mux,
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BaseContext: func(config net.Listener) context.Context {
			return ctx
		},
	}

	return server.ListenAndServe()
}

func main() {
	var host string

	flag.StringVar(&host, "h", "127.0.0.1", "host")
	flag.Parse()

	log.SetFlags(0)

	if flag.NArg() != 2 {
		log.Fatalf("usage: %s <descriptor-set> <script>", os.Args[0])
	}

	port := 8888
	setPath := flag.Arg(0)
	scriptPath := flag.Arg(1)

	log.Printf("Starting listener on %s:%d...", host, port)

	if err := run(setPath, scriptPath, host, port); err != nil {
		panic(err)
	}
}

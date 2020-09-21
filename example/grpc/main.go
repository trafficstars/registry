package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"

	"github.com/trafficstars/registry"
	"github.com/trafficstars/registry/example/grpc/server"
	registry_balancer "github.com/trafficstars/registry/net/balancer"
	registry_grpc "github.com/trafficstars/registry/net/grpc"
)

var (
	flagRegistryConnect  = flag.String("registry", "http://localhost:8500?dc=default&refresh_interval=5", "connection to the registry")
	flagGRPCServerListen = flag.String("server", ":50051", "Server connection")
)

const (
	registryGRPCID = "testgrpc"
	address        = "registry://" + registryGRPCID
)

type serverObject struct{}

func (s serverObject) Ping(ctx context.Context, req *server.Request) (*server.Response, error) {
	return &server.Response{Msg: "pong->" + req.Msg}, nil
}

func main() {
	flag.Parse()

	wait := make(chan bool)

	// Init discovery service
	log.Println("Init registry", *flagRegistryConnect)
	reg, err := registry.New(*flagRegistryConnect, os.Args)
	if err != nil {
		log.Fatalf("failed init registry: %v", err)
	}
	discovery := reg.Discovery()

	// Register GRPC balancer and resolver
	registry_balancer.Init(registry_balancer.RoundRobinStrategy, discovery)
	balancer.Register(registry_grpc.NewBalancerBuilder("registry"))
	resolver.Register(registry_grpc.NewResolveBuilder("registry", discovery))
	resolver.SetDefaultScheme("registry")

	// Register GRPC server
	go func() {
		log.Println("Run GRPC server", *flagGRPCServerListen)
		lis, err := net.Listen("tcp", *flagGRPCServerListen)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		srv := grpc.NewServer()
		server.RegisterTestServer(srv, serverObject{})

		wait <- true

		if err := srv.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-wait

	log.Println("Register in service descovery")

	// Register server
	err = discovery.Register(registry.ServiceOptions{
		ID:      registryGRPCID,
		Name:    registryGRPCID,
		Address: *flagGRPCServerListen,
		Tags:    []string{"test"},
	})
	if err != nil {
		log.Fatalf("register service: %v", err)
	}
	defer discovery.Deregister(registryGRPCID)
	// time.Sleep(time.Second * 3)
	registry_balancer.Default().Refresh()

	// init GRPC connection
	{
		log.Println("Init GRPC client")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		conn, err := grpc.DialContext(ctx, address+*flagGRPCServerListen, grpc.WithInsecure(), grpc.WithBalancerName("registry"))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()

		client := server.NewTestClient(conn)

		for i := 0; i < 5; i++ {
			log.Println("Send request", i+1)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			r, err := client.Ping(ctx, &server.Request{Msg: "test"})
			if err != nil {
				log.Fatalf("could not test: %v", err)
			}
			log.Printf("Test: %s", r.GetMsg())
			time.Sleep(time.Second * 1)
		}
	}
}

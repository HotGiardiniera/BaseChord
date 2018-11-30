package main

import (
	"flag"
	//"fmt"
	"log"
	"strconv"
	"strings"

	context "golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/nyu-distributed-systems-fa18/BaseChord/pb"
)

func Connect(server string) pb.FileSystemClient {
	// Connect to the server. We use WithInsecure since we do not configure https in this class.
	log.Printf("Connecting to  %v", server)
	conn, err := grpc.Dial(server, grpc.WithInsecure())
	//Ensure connection did not fail.
	if err != nil {
		log.Fatalf("Failed to dial GRPC server %v", err)
	}
	// Create a FileSystem client
	return pb.NewFileSystemClient(conn)
}

func handleResult(res *pb.Result, Server string) {
    var message string
    if notFound := res.GetNotFound(); notFound != nil {
        message = "File not found!"
    }

    log.Printf("%v respose: %v", Server, message)
}

func checkRedirect(res *pb.Result) (bool, string) {
	redirectTo := ""
	if redirect := res.GetRedirect(); redirect != nil {
		redirectIP := strings.Split(redirect.Server, ":")
		port, _ := strconv.Atoi(redirectIP[1])
		sPort := strconv.Itoa(port - 1)
		redirectIP[1] = sPort
		redirectTo = strings.Join(redirectIP, ":")
	}
	return redirectTo != "", redirectTo
}

func basicGet(fs pb.FileSystemClient, Server string) {
	// Request value for Chris
	req := &pb.FileGet{Name: "Chris"}
	res, err := fs.Get(context.Background(), req)
	if err != nil {
		log.Fatalf("Request error %v", err)
	}
	redirect, redirectIP := checkRedirect(res)
	if redirect {
		log.Printf("File at a differnt node. Redirecting to: %v", redirectIP)
		fs = Connect(redirectIP)
		basicGet(fs, redirectIP)
	} else {
        handleResult(res, Server)
	}
}

func main() {
	// Take endpoint as input
	var call string
	var endpoint string

	flag.StringVar(&endpoint, "endpoint", "127.0.0.1:3000", "Client endpoint")
	flag.StringVar(&call, "call", "main", "Choose single functions to run or OG main")
	// flag.Usage = usage
	flag.Parse()
	// If there is no endpoint fail
	// endpoint := flag.Args()[0]

	// if flag.NArg() == 0 {
	//  flag.Usage()
	//  os.Exit(1)
	// }

	// Create a FileSystem client
	fs := Connect(endpoint)

	var fnc func(pb.FileSystemClient, string)

	fnc = basicGet

	// TODO allow certain calls from the command line
	// switch call {
	// case "set":
	//     log.Printf("set")
	//     fnc = basic_set
	// case "get":
	//     log.Printf("get")
	//     fnc = basic_get
	// default:
	//     fnc = originalMain
	// }

	fnc(fs, endpoint)
}

package main

import (
	"flag"
	"fmt"
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

func handleResult(res *pb.Result, file, Server string) {
	var message string
	if notFound := res.GetNotFound(); notFound != nil {
		message = fmt.Sprintf("File %v not found!", file)
	}
	if stored := res.GetSuccess(); stored != nil {
		message = fmt.Sprintf("File %v successfully stored/deleted!", file)
	}
	if found := res.GetData(); found != nil {
		message = fmt.Sprintf("File %v found! Data: %v", file, found.Data)
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

func Get(fs pb.FileSystemClient, fileName, Server string) {
	// Request value for Chris
	req := &pb.FileGet{Name: fileName}
	res, err := fs.Get(context.Background(), req)
	log.Printf("Getting file: %v", fileName)
	if err != nil {
		log.Fatalf("Request error %v", err)
	}
	redirect, redirectIP := checkRedirect(res)
	if redirect {
		log.Printf("File at a differnt node. Redirecting to: %v", redirectIP)
		fs = Connect(redirectIP)
		Get(fs, fileName, redirectIP)
	} else {
		handleResult(res, fileName, Server)
	}
}

func Store(fs pb.FileSystemClient, fileName, Server string) {
	// Request value for Chris
	req := &pb.FileStore{Name: fileName, Data: &pb.Data{Data: fileName}}
	res, err := fs.Store(context.Background(), req)
	log.Printf("Soring file: %v", fileName)
	if err != nil {
		log.Fatalf("Request error %v", err)
	}
	redirect, redirectIP := checkRedirect(res)
	if redirect {
		log.Printf("File need to be stored at a differnt node. Redirecting to: %v", redirectIP)
		fs = Connect(redirectIP)
		Store(fs, fileName, redirectIP)
	} else {
		handleResult(res, fileName, Server)
	}
}

func Delete(fs pb.FileSystemClient, fileName, Server string) {
	// Request value for Chris
	req := &pb.FileDelete{Name: fileName}
	res, err := fs.Delete(context.Background(), req)
	log.Printf("Deleting file: %v", fileName)
	if err != nil {
		log.Fatalf("Request error %v", err)
	}
	redirect, redirectIP := checkRedirect(res)
	if redirect {
		log.Printf("File need to be stored at a differnt node. Redirecting to: %v", redirectIP)
		fs = Connect(redirectIP)
		Delete(fs, fileName, redirectIP)
	} else {
		handleResult(res, fileName, Server)
	}
}

func main() {
	// Take endpoint as input
	var call string
	var fileName string
	var endpoint string

	flag.StringVar(&endpoint, "endpoint", "127.0.0.1:3000", "Client endpoint")
	flag.StringVar(&call, "call", "", "Choose single functions to run or OG main")
	flag.StringVar(&fileName, "file", "default", "File name to get, insert, or delete")
	flag.Parse()

	// Create a FileSystem client
	fs := Connect(endpoint)

	var fnc func(pb.FileSystemClient, string, string)

	switch call {
	case "get":
		log.Printf("get")
		fnc = Get
	case "store":
		log.Printf("store")
		fnc = Store
	case "delete":
		log.Printf("delete")
		fnc = Delete
	default:
		fnc = Get
	}

	fnc(fs, fileName, endpoint)
}

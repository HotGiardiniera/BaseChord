package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	// "math/rand"
	// "time"

	context "golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/nyu-distributed-systems-fa18/BaseChord/pb"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Connect to desired port or chord peer
func Connect(server string) pb.FileSystemClient {
	// If user passed in chord client:
	if strings.Contains(server, "chord") {
		config, err := clientcmd.BuildConfigFromFlags("", "/home/vagrant/.kube/config")
		if err != nil {
			panic(err.Error())
		}
		// creates the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		for _, pod := range pods.Items {
			if pod.Name == server {
				log.Printf("Connecting to %v: %v", server, pod.Status.PodIP)
				server = pod.Status.PodIP + ":3000"
				break
			}
		}
	} else {
		log.Printf("Connecting to %v", server)
	}

	// Connect to the server. We use WithInsecure since we do not configure https in this class.
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
		message = fmt.Sprintf("File \"%v\" not found!", file)
	}
	if stored := res.GetSuccess(); stored != nil {
		message = fmt.Sprintf("File \"%v\" successfully stored/deleted!", file)
	}
	if found := res.GetData(); found != nil {
		message = fmt.Sprintf("File \"%v\" found! Data: %v", file, found.Data)
	}

	log.Printf("%v response: %v", Server, message)
}

func checkRedirect(res *pb.Result) (bool, string) {
	redirectTo := ""
	if redirect := res.GetRedirect(); redirect != nil {
		redirectIP := strings.Split(redirect.Server, ":")
		if strings.Contains(redirect.Server, "chord") {
			redirectTo = redirectIP[0]
		} else {
			port, _ := strconv.Atoi(redirectIP[1])
			sPort := strconv.Itoa(port - 1)
			redirectIP[1] = sPort
			redirectTo = strings.Join(redirectIP, ":")
		}
	}
	return redirectTo != "", redirectTo
}

// Get sends a file retrieval request to chord ring
func Get(fs pb.FileSystemClient, fileName, Server string) {
	// Request value for Chris
	req := &pb.FileGet{Name: fileName}
	res, err := fs.Get(context.Background(), req)
	log.Printf("Getting file: \"%v\"", fileName)
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

// Store sends a file storage request to chord ring
func Store(fs pb.FileSystemClient, fileName, Server string) {
	// Request value for Chris
	req := &pb.FileStore{Name: fileName, Data: &pb.Data{Data: fileName}}
	res, err := fs.Store(context.Background(), req)
	log.Printf("Storing file: \"%v\"", fileName)
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

// Delete sends a file removal request to chord ring
func Delete(fs pb.FileSystemClient, fileName, Server string) {
	// Request value for Chris
	req := &pb.FileDelete{Name: fileName}
	res, err := fs.Delete(context.Background(), req)
	log.Printf("Deleting file: \"%v\"", fileName)
	if err != nil {
		log.Fatalf("Request error %v", err)
	}
	redirect, redirectIP := checkRedirect(res)
	if redirect {
		log.Printf("File at a differnt node. Redirecting to: %v", redirectIP)
		fs = Connect(redirectIP)
		Delete(fs, fileName, redirectIP)
	} else {
		handleResult(res, fileName, Server)
	}
}

// TODO make random filename generator
// TODO for client TODAY make a bunch of calls, add 8  servers, make a few more stores/get add 8 more, get metrics

//------------------------------- LOCAL TESTS --------------------------------//
func exercise_local(portString string) {
	files := make([]string, 100) // Will hash nearish to each other
	for i := 0; i < 100; i++ {
		files[i] = fmt.Sprintf("File%v", i)
	}

	ports := strings.Split(portString, ",")
	// First store all files
	endpoint := fmt.Sprintf("stretch:3%03v", ports[0])

	log.Printf("%v\n", endpoint)

	fs := Connect(endpoint)

	for _, file := range files {
		Store(fs, file, endpoint)
	}

	for _, port := range ports[1:] {
		endpoint = fmt.Sprintf("stretch:3%03v", port)
		fs = Connect(endpoint)
		for _, file := range files {
			Get(fs, file, endpoint)
		}
	}

}

//----------------------------------------------------------------------------//
//-------------------------------- KUBE TESTS --------------------------------//

func exercise_kube(numFiles int) {
	files := make([]string, numFiles) // Will hash nearish to each other
	for i := 0; i < numFiles; i++ {
		files[i] = fmt.Sprintf("File%v", i)
	}

	config, err := clientcmd.BuildConfigFromFlags("", "/home/vagrant/.kube/config")
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	var server string
	servers := make([]string, 0)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "chord"){
			// log.Printf("Server name %v : %v", pod.Name, pod.Status.PodIP)
			server = pod.Status.PodIP + ":3000"
			servers = append(servers, server)
		}	
	}


	log.Printf("Storing files at %v", servers[0])
	fs := Connect(servers[0])

	for _, file := range files {
		Store(fs, file, servers[0])
	}

	for _, server := range servers[1:] {
		log.Printf("Connecting to %v", server)
		fs = Connect(server)
		for _, file := range files {
			Get(fs, file, server)
		}
	}

}

//----------------------------------------------------------------------------//

func main() {
	// Take endpoint as input
	var call string
	var fileName string
	var localPorts string

	//Local exercising
	var endpoint string
	flag.StringVar(&localPorts, "local_ports", "", "comma separated list of client ports, 1,2,3,4. If active will ignore any other flags")

	//Kube exercising
	var kube bool
	var files int
	flag.BoolVar(&kube, "kube", false, "Store/Get 100 files from kube")
	flag.IntVar(&files, "files", 100, "Number of files to generate")

	flag.StringVar(&endpoint, "endpoint", "127.0.0.1:3000", "Client endpoint")
	flag.StringVar(&call, "call", "", "Choose single functions to run or OG main")
	flag.StringVar(&fileName, "file", "default", "File name to get, insert, or delete")
	flag.Parse()

	// Create a FileSystem client
	fs := Connect(endpoint)

	if localPorts != "" {
		exercise_local(localPorts)
		return
	} else if kube {
		exercise_kube(files)
		return
	}

	var fnc func(pb.FileSystemClient, string, string)

	switch call {
	case "get":
		fnc = Get
	case "store":
		fnc = Store
	case "delete":
		fnc = Delete
	default:
		fnc = Get
	}

	fnc(fs, fileName, endpoint)
}

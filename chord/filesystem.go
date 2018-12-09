package main

import (
	"log"
	"strconv"

	context "golang.org/x/net/context"

	"github.com/nyu-distributed-systems-fa18/BaseChord/pb"
)

// InputChannelType is struct for data to send over channel
type InputChannelType struct {
	command  pb.Command
	response chan pb.Result
}

// FileSystem is struct for key value stores.
type FileSystem struct {
	C          chan InputChannelType
	fileSystem map[string]string
}

// Get file by name
func (fs *FileSystem) Get(ctx context.Context, file *pb.FileGet) (*pb.Result, error) {
	// Create a channel
	c := make(chan pb.Result)
	// Create a request
	r := pb.Command{Operation: pb.Op_GET, Arg: &pb.Command_Get{Get: file}}
	// Send request over the channel
	fs.C <- InputChannelType{command: r, response: c}
	log.Printf("Waiting for file retrieval response...")
	result := <-c
	return &result, nil
}

// Store file
func (fs *FileSystem) Store(ctx context.Context, file *pb.FileStore) (*pb.Result, error) {
	// Create a channel
	c := make(chan pb.Result)
	// Create a request
	r := pb.Command{Operation: pb.Op_STORE, Arg: &pb.Command_Store{Store: file}}
	// Send request over the channel
	fs.C <- InputChannelType{command: r, response: c}
	log.Printf("Waiting for file store response...")
	result := <-c
	return &result, nil
}

// Delete a file
func (fs *FileSystem) Delete(ctx context.Context, file *pb.FileDelete) (*pb.Result, error) {
	// Create a channel
	c := make(chan pb.Result)
	// Create a request
	r := pb.Command{Operation: pb.Op_DELETE, Arg: &pb.Command_Delete{Delete: file}}
	// Send request over the channel
	fs.C <- InputChannelType{command: r, response: c}
	log.Printf("Waiting for file deletion response...")
	result := <-c
	return &result, nil
}

// GetInternal : Used internally to generate a result for a get request. This
//function assumes that it is called from a single thread of
//execution, and hence does not handle races.
func (fs *FileSystem) GetInternal(file string) pb.Result {
	if d, ok := fs.fileSystem[file]; ok {
		return pb.Result{Result: &pb.Result_Data{Data: &pb.Data{Data: d}}}
	}
	return pb.Result{Result: &pb.Result_NotFound{NotFound: &pb.FileNotFound{}}}
}

// StoreInternal : Used internally to set and generate an appropriate result. This
//function assumes that it is called from a single
//thread of execution and hence does not handle race conditions.
func (fs *FileSystem) StoreInternal(f string, d *pb.Data) pb.Result {
	hashedData := strconv.FormatUint(generateIDFromIP(d.Data), 10)
	log.Printf(yellow("Attempting to store %s at %s"), f, hashedData)
	fs.fileSystem[f] = hashedData
	return pb.Result{Result: &pb.Result_Success{Success: &pb.Success{}}}
}

// DeleteInternal : Used internally, this function clears a kv store. Assumes no
//racing calls.
func (fs *FileSystem) DeleteInternal(f string) pb.Result {
	_, ok := fs.fileSystem[f]
	if ok {
		delete(fs.fileSystem, f)
		return pb.Result{Result: &pb.Result_Success{Success: &pb.Success{}}}
	}
	return pb.Result{Result: &pb.Result_NotFound{NotFound: &pb.FileNotFound{}}}
}

// MoveInternal : Used internally to flag any files that should be passed to a
//predecessor
func (fs *FileSystem) MoveInternal(myID, predecessorID uint64) map[string]string {
	var filesToMove map[string]string
	filesToMove = make(map[string]string)
	var dataToCompare uint64
	for name, data := range fs.fileSystem {
		log.Printf("Looking if we should move file %v-%v to %v", name, data, predecessorID)
		dataToCompare, _ = strconv.ParseUint(data, 10, 64)
		// If file hashed key is less than predecessorID, makre file for
		//move to our predecessor
		if dataToCompare < predecessorID && predecessorID < myID {
			log.Printf(red("Yes we should!"))
			filesToMove[name] = data
		}
	}
	return filesToMove
}

// HandleCommand is interface to raft part of the server
func (fs *FileSystem) HandleCommand(op InputChannelType) {
	switch c := op.command; c.Operation {
	case pb.Op_GET:
		arg := c.GetGet()
		result := fs.GetInternal(arg.Name)
		op.response <- result
	case pb.Op_STORE:
		arg := c.GetStore()
		result := fs.StoreInternal(arg.Name, arg.Data)
		op.response <- result
	case pb.Op_DELETE:
		arg := c.GetDelete()
		result := fs.DeleteInternal(arg.Name)
		op.response <- result
	default:
		// Sending a blank response to just free things up, but we don't know how to make progress here.
		op.response <- pb.Result{}
		log.Fatalf("Unrecognized operation %v", c)
	}
}

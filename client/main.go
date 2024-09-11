package main

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lgfa29/frpc/cpstream"
	"github.com/loopholelabs/frisbee-go"
)

func main() {
	c, err := cpstream.NewClient(nil, nil)
	if err != nil {
		panic(err)
	}

	err = c.Connect("127.0.0.1:3333")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	log.Println("Connected to server on 127.0.0.1:3333")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	done := make(chan any)
	go func() {
		<-stop
		close(done)
	}()

	stream, err := c.ControlPlaneStream.Stream(context.Background(), nil)

	errCh := make(chan error)
	cmdCh := make(chan *cpstream.ControlPlaneCommand)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}

			res, err := stream.Recv()
			if err != nil {
				errCh <- err
			} else {
				cmdCh <- res
			}
		}
	}()

	for {
		select {
		case <-done:
			log.Println("Closing receiver")
			return

		case err := <-errCh:
			if errors.Is(err, io.EOF) || errors.Is(err, frisbee.StreamClosed) {
				log.Println(err)
				return
			}
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				continue
			}

		case cmd := <-cmdCh:
			log.Printf("Received command %s\n", cmd.Uuid)

			if cmd.CreateInstanceCmd.Name != "" {
				go handleCreateInstanceCmd(stream, cmd)
			} else {
				go handleListInstanceCmd(stream, cmd)
			}
		}
	}
}

func handleListInstanceCmd(stream *cpstream.StreamClient, cmd *cpstream.ControlPlaneCommand) {
	sendResponse(stream, &cpstream.ControlPlaneResponse{
		Uuid: cmd.Uuid,
		ListInstancesResp: &cpstream.ListInstancesResponse{
			Instances: []string{"a", "b"},
		},
	})
}

func handleCreateInstanceCmd(stream *cpstream.StreamClient, cmd *cpstream.ControlPlaneCommand) {
	sendResponse(stream, &cpstream.ControlPlaneResponse{
		Uuid: cmd.Uuid,
		CreateInstanceResp: &cpstream.CreateInstanceResponse{
			Name: cmd.CreateInstanceCmd.Name,
		},
	})
}

func sendResponse(stream *cpstream.StreamClient, res *cpstream.ControlPlaneResponse) {
	log.Printf("Sending response %s\n", res.Uuid)
	err := stream.Send(res)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return
	}
}

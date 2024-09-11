package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/lgfa29/frpc/cpstream"
	"github.com/lgfa29/frpc/pkg/future"
	"github.com/loopholelabs/frisbee-go"
)

type cpReq struct {
	*cpstream.ControlPlaneCommand
	f *future.Future
}

type cpSvc struct {
	reqCh chan *cpReq

	pendingReqs     map[string]*future.Future
	pendingReqsLock sync.RWMutex
}

func (cp *cpSvc) Stream(ctx context.Context, srv *cpstream.StreamServer) error {
	errCh := make(chan error)
	resCh := make(chan *cpstream.ControlPlaneResponse)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			res, err := srv.Recv()
			if err != nil {
				errCh <- err
			} else {
				resCh <- res
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-errCh:
			if errors.Is(err, io.EOF) {
				return srv.CloseSend()
			}
			if errors.Is(err, frisbee.StreamClosed) {
				return nil
			}
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				continue
			}

		case req := <-cp.reqCh:
			go func() {
				cp.pendingReqsLock.Lock()
				defer cp.pendingReqsLock.Unlock()

				log.Printf("Sending request %s\n", req.Uuid)
				err := srv.Send(req.ControlPlaneCommand)
				if err != nil {
					req.f.Fail(err)
					return
				}
				cp.pendingReqs[req.Uuid] = req.f
			}()

		case res := <-resCh:
			go func() {
				cp.pendingReqsLock.RLock()
				defer cp.pendingReqsLock.RUnlock()

				log.Printf("Received response %s\n", res.Uuid)
				f, ok := cp.pendingReqs[res.Uuid]
				if !ok {
					log.Printf("Response to unknown request %s\n", res.Uuid)
					return
				}

				if res.Err != "" {
					f.Fail(errors.New(res.Err))
					return
				}
				f.Fulfill(res)
			}()
		}
	}
}

func (cp *cpSvc) ListIntances() ([]string, error) {
	f := cp.makeRequest(&cpstream.ControlPlaneCommand{
		ListInstancesCmd: &cpstream.ListInstancesCommand{},
	})

	r, err := f.Wait()
	if err != nil {
		return nil, err
	}

	res, _ := r.(*cpstream.ControlPlaneResponse)
	return res.ListInstancesResp.Instances, nil
}

func (cp *cpSvc) CreateInstance() (string, error) {
	f := cp.makeRequest(&cpstream.ControlPlaneCommand{
		CreateInstanceCmd: &cpstream.CreateInstanceCommand{
			Name: "test",
		},
	})

	r, err := f.Wait()
	if err != nil {
		return "", err
	}

	res, _ := r.(*cpstream.ControlPlaneResponse)
	return res.CreateInstanceResp.Name, nil
}

func (cp *cpSvc) makeRequest(cmd *cpstream.ControlPlaneCommand) *future.Future {
	cmd.Uuid = uuid.New().String()
	f := future.New()

	cp.reqCh <- &cpReq{
		f:                   f,
		ControlPlaneCommand: cmd,
	}

	return f
}

func main() {
	cp := &cpSvc{
		reqCh:       make(chan *cpReq),
		pendingReqs: make(map[string]*future.Future),
	}

	frpcServer, err := cpstream.NewServer(cp, nil, nil)
	if err != nil {
		panic(err)
	}

	go func() {
		log.Println("RPC server listening on :3333")
		err = frpcServer.Start(":3333")
		if err != nil {
			panic(err)
		}
	}()

	http.HandleFunc("/instances", func(w http.ResponseWriter, r *http.Request) {
		var err error
		var res any

		switch r.Method {
		case http.MethodGet:
			res, err = cp.ListIntances()
		case http.MethodPost:
			res, err = cp.CreateInstance()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	})

	log.Println("HTTP server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

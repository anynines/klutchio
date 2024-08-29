/*
Copyright 2024 Klutch Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package healthz

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Server struct {
	router               *mux.Router
	listener             net.Listener
	backendHealthHandler backendHealthHandler
}

func NewServer(mgr manager.Manager, listenAddress string) (*Server, error) {
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return nil, err
	}

	server := &Server{
		router:   mux.NewRouter(),
		listener: listener,
		backendHealthHandler: backendHealthHandler{
			client: mgr.GetClient(),
		},
	}

	server.router.HandleFunc("/healthz", handleHealthz)
	server.router.Handle("/backend-health", server.backendHealthHandler)

	return server, nil
}

func (s *Server) Start(ctx context.Context) error {
	httpServer := &http.Server{
		Handler:     s.router,
		ReadTimeout: 20 * time.Second,
	}
	go func() {
		<-ctx.Done()
		err := httpServer.Close()
		if err != nil {
			fmt.Printf("error closing http server: %v", err)
		}
	}()

	go func() {
		err := httpServer.Serve(s.listener)
		if err != nil {
			fmt.Printf("error serving http endpoint: %v", err)
		}
	}()

	return nil
}

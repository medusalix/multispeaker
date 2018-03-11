/*
 * Copyright (C) 2018 Medusalix
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"github.com/medusalix/multispeaker/cli"
	"github.com/medusalix/multispeaker/log"
	"github.com/medusalix/multispeaker/network"
	"net"
	"os"
)

const defaultControlPort = 12345
const defaultStreamPort = 12346

// Can be specified using linker flag "-X"
var defaultClientAddr string

func main() {
	// Hide if default address is specified
	if defaultClientAddr != "" && len(os.Args) == 1 {
		cli.HideConsole()
	}

	cli.Writeln("multispeaker v1.0.1 ©Severin v. W.\n")

	logLevel := flag.String("log", "info", "Desired log level")
	controlPort := flag.Int("control-port", defaultControlPort, "Port for the control connection")
	streamPort := flag.Int("stream-port", defaultStreamPort, "Port for the stream connection")
	server := flag.Bool("server", false, "Start as server")
	client := flag.String("client", defaultClientAddr, "Address to connect the client to")

	flag.Parse()

	log.Init(cli.Writef, *logLevel)

	controlAddr := &net.TCPAddr{Port: *controlPort}
	streamAddr := &net.TCPAddr{Port: *streamPort}

	if *server {
		cli.SetPrompt("> ")

		server := network.NewServer(controlAddr, streamAddr)
		err := server.Run()

		if err != nil {
			cli.Writeln("Error starting server:", err)

			return
		}

		cli.HandleCommands(server)
	} else if *client != "" {
		addr, err := net.ResolveIPAddr("ip", *client)

		if err != nil {
			cli.Writeln("Error resolving address:", err)

			return
		}

		controlAddr.IP = addr.IP
		streamAddr.IP = addr.IP

		client := network.NewClient(controlAddr, streamAddr)
		err = client.Run()

		if err != nil {
			cli.Writeln("Error starting client:", err)

			return
		}
	} else {
		flag.PrintDefaults()
	}
}

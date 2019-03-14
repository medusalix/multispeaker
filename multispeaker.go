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
	"net"
	"os"

	"github.com/medusalix/multispeaker/cli"
	"github.com/medusalix/multispeaker/log"
	"github.com/medusalix/multispeaker/network"
)

const (
	defaultControlPort = 12345
	defaultStreamPort  = 12346
)

// Can be specified using linker flag "-X"
var defaultClientAddr string

func main() {
	logLevel := flag.String("log", "info", "Desired log level")
	hide := flag.Bool("hide", false, "Hide the console")
	controlPort := flag.Int("control-port", defaultControlPort, "Port for the control connection")
	streamPort := flag.Int("stream-port", defaultStreamPort, "Port for the stream connection")
	server := flag.Bool("server", false, "Start as server")
	client := flag.String("client", defaultClientAddr, "Address to connect the client to")

	flag.Parse()

	log.Init(cli.Writef, *logLevel)

	// Hide if default address is specified or flag is set
	if defaultClientAddr != "" && len(os.Args) == 1 {
		cli.HideConsole(true)
	} else if *hide {
		cli.HideConsole(false)
	}

	cli.Writeln("multispeaker v1.0.3 ©Severin v. W.")
	cli.Writeln()

	controlAddr := &net.TCPAddr{Port: *controlPort}
	streamAddr := &net.TCPAddr{Port: *streamPort}

	if *server {
		cli.Prompt = "> "

		server := network.NewServer(controlAddr, streamAddr)

		if err := server.Start(); err != nil {
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

		if err := client.Start(); err != nil {
			cli.Writeln("Error starting client:", err)

			return
		}
	} else {
		flag.PrintDefaults()
	}
}

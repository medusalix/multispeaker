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

package network

import (
	"net"

	"github.com/medusalix/multispeaker/log"
)

type endpoint struct {
	ip            net.IP
	name          string
	statusChanged statusCallback
	control       *protocol
	stream        *protocol
	samples       chan []byte
}

type statusCallback func(endpoint *endpoint, connected bool)

func newEndpoint(conn net.Conn, statusChanged statusCallback) *endpoint {
	endpoint := &endpoint{
		ip:            conn.RemoteAddr().(*net.TCPAddr).IP,
		control:       newProtocol(conn),
		statusChanged: statusChanged,
	}

	go endpoint.listen()

	return endpoint
}

func (e *endpoint) connectStream(conn net.Conn) {
	e.stream = newProtocol(conn)
	e.samples = make(chan []byte)
}

func (e *endpoint) disconnectStream() error {
	if e.stream == nil {
		return nil
	}

	err := e.stream.close()
	e.stream = nil

	return err
}

func (e *endpoint) streamSamples(samples []byte) error {
	if e.stream == nil {
		return nil
	}

	_, err := e.stream.sendRaw(samples)

	if err != nil {
		e.stream = nil
	}

	return err
}

func (e *endpoint) preparePlayback(sampleRate int) error {
	return e.control.send(&preparePacket{
		sampleRate: sampleRate,
	})
}

func (e *endpoint) changeVolume(volume int) error {
	return e.control.send(&volumePacket{
		volume: volume,
	})
}

func (e *endpoint) listen() {
	for {
		packet, err := e.control.receive()

		if err != nil {
			if err := e.disconnectStream(); err != nil {
				log.Error("Error disconnecting stream: ", err)
			}

			// Notify server of disconnect
			e.statusChanged(e, false)

			return
		}

		switch p := packet.(type) {
		case *announcePacket:
			e.name = p.name
			e.statusChanged(e, true)
		}
	}
}

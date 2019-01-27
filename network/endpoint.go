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
	"errors"
	"net"

	"github.com/medusalix/multispeaker/log"
)

type Endpoint struct {
	Ip   net.IP
	Name string

	control *Protocol
	stream  *Protocol

	sampleRate int
	samples    chan []byte

	disconnect chan *Endpoint
	close      chan bool
}

func NewEndpoint(conn net.Conn, disconnect chan *Endpoint) *Endpoint {
	endpoint := &Endpoint{
		Ip: conn.RemoteAddr().(*net.TCPAddr).IP,

		control: NewProtocol(conn),
		samples: make(chan []byte),

		disconnect: disconnect,
		close:      make(chan bool),
	}

	go endpoint.listen()

	return endpoint
}

func (e *Endpoint) EnableStreaming(streamConn net.Conn) bool {
	if e.stream == nil {
		e.stream = NewProtocol(streamConn)

		go e.sendStream()

		return true
	}

	return false
}

func (e *Endpoint) PreparePlayback(sampleRate int) error {
	if sampleRate < 0 {
		return errors.New("sample rate should be >= 0")
	} else if sampleRate > 48000 {
		return errors.New("sample rate should be <= 48000")
	}

	return e.control.Send(&PreparePacket{
		SampleRate: uint16(sampleRate),
	})
}

func (e *Endpoint) ChangeVolume(volume int) error {
	if volume < 0 {
		return errors.New("volume should be >= 0")
	} else if volume > 100 {
		return errors.New("volume should be <= 100")
	}

	return e.control.Send(&ControlPacket{
		Volume: uint8(volume),
	})
}

func (e *Endpoint) StreamSamples(samples []byte) error {
	if e.stream != nil {
		e.samples <- samples
	}

	return nil
}

func (e *Endpoint) listen() {
	for {
		packet, err := e.control.Receive()

		if err != nil {
			// Notify server of disconnect
			e.disconnect <- e

			// Close stream connection
			e.stream = nil
			e.close <- true

			log.Infof("Endpoint '%s' disconnected", e.Name)

			return
		}

		switch p := packet.(type) {
		case *AnnouncePacket:
			e.Name = string(p.Name)

			log.Infof("Endpoint '%s' connected", e.Name)
		}
	}
}

func (e *Endpoint) sendStream() {
	for {
		select {
		case <-e.close:
			return
		case samples := <-e.samples:
			// Ignore error, we exit manually
			e.stream.SendRaw(samples)
		}
	}
}

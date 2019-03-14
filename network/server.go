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
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/medusalix/multispeaker/audio"
	"github.com/medusalix/multispeaker/log"
)

const streamReadyTimeout = time.Second * 5

// Server is used to accept new clients and stream music
type Server struct {
	controlAddr     *net.TCPAddr
	streamAddr      *net.TCPAddr
	controlListener net.Listener
	streamListener  net.Listener
	endpoints       map[string]*endpoint
	mutex           sync.RWMutex
	music           *audio.Music
	streamReady     chan bool
	streaming       bool
}

// NewServer constructs a new server
func NewServer(controlAddr *net.TCPAddr, streamAddr *net.TCPAddr) *Server {
	return &Server{
		controlAddr: controlAddr,
		streamAddr:  streamAddr,
		endpoints:   make(map[string]*endpoint),
		music:       audio.NewMusic(),
		streamReady: make(chan bool),
	}
}

// Start starts the server
func (s *Server) Start() error {
	var err error
	s.controlListener, err = net.ListenTCP("tcp", s.controlAddr)

	if err != nil {
		return err
	}

	s.streamListener, err = net.ListenTCP("tcp", s.streamAddr)

	if err != nil {
		return err
	}

	go s.listenControl()
	go s.listenStream()

	return nil
}

// GetConnectedUsers returns a list of the currently connected users
func (s *Server) GetConnectedUsers() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	users := make([]string, 0, len(s.endpoints))

	for _, endpoint := range s.endpoints {
		users = append(users, endpoint.name)
	}

	return users
}

// PlayMusic initiates the music playback
func (s *Server) PlayMusic(filePath string) error {
	if s.streaming {
		return errors.New("music is already playing")
	}

	sampleRate, err := s.music.Load(filePath)

	if err != nil {
		return err
	}

	log.Debug("Preparing music playback")

	s.allEndpoints(func(endpoint *endpoint) error {
		return endpoint.preparePlayback(sampleRate)
	}, func(endpoint *endpoint, err error) {
		log.Errorf("Unable to start playback for '%s': %s", endpoint.name, err)
	})

	select {
	case <-s.streamReady:
	case <-time.After(streamReadyTimeout):
		log.Info("Waiting for endpoints timed out")
	}

	s.streaming = true
	go s.streamMusic()

	return nil
}

// StopMusic stops the music playback
func (s *Server) StopMusic() error {
	if !s.streaming {
		return errors.New("music is currently not playing")
	}

	err := s.music.Close()

	s.allEndpoints(func(endpoint *endpoint) error {
		return endpoint.preparePlayback(0)
	}, func(endpoint *endpoint, err error) {
		log.Errorf("Unable to stop playback for '%s': %s", endpoint.name, err)
	})

	s.allEndpoints(func(endpoint *endpoint) error {
		return endpoint.disconnectStream()
	}, func(endpoint *endpoint, err error) {
		log.Errorf("Error disconnecting stream of '%s': %s", endpoint.name, err)
	})

	s.streaming = false

	return err
}

// SetVolume sets the volume of the specified user (or all users)
func (s *Server) SetVolume(user string, volume int) error {
	found := false

	s.allEndpoints(func(endpoint *endpoint) error {
		if endpoint.name != user && user != "all" {
			return nil
		}

		found = true

		return endpoint.changeVolume(volume)
	}, func(endpoint *endpoint, err error) {
		log.Debugf("Error changing volume of '%s'", endpoint.name)
	})

	if !found {
		return fmt.Errorf("no user with name '%s' found", user)
	}

	return nil
}

func (s *Server) listenControl() {
	for {
		conn, err := s.controlListener.Accept()

		if err != nil {
			log.Error("Unable to accept control client: ", err)

			continue
		}

		ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()

		s.mutex.Lock()

		// Check if client is already connected
		if _, ok := s.endpoints[ip]; ok {
			log.Debugf("Duplicate connection from client with IP '%s'", ip)
			conn.Close()
		} else {
			// New endpoint connected
			log.Debugf("New control connection from '%s'", ip)
			s.endpoints[ip] = newEndpoint(conn, s.handleStatusChange)
		}

		s.mutex.Unlock()
	}
}

func (s *Server) listenStream() {
	for {
		conn, err := s.streamListener.Accept()

		if err != nil {
			log.Error("Unable to accept stream client: ", err)

			continue
		}

		ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()

		s.mutex.RLock()

		// Check if there's a control connection available
		if endpoint, ok := s.endpoints[ip]; ok {
			if s.streaming {
				log.Debugf("Client '%s' connected while streaming", ip)
				conn.Close()
			} else {
				log.Debugf("New stream connection from '%s'", ip)
				endpoint.connectStream(conn)
				s.checkStreamReady()
			}
		} else {
			// No control connection
			conn.Close()
		}

		s.mutex.RUnlock()
	}
}

func (s *Server) checkStreamReady() {
	streamReady := true

	// Check if all endpoints are ready
	for _, endpoint := range s.endpoints {
		if endpoint.stream == nil {
			streamReady = false

			break
		}
	}

	// Start music if all endpoints are ready
	if streamReady {
		s.streamReady <- true
	}
}

func (s *Server) streamMusic() {
	for {
		samples, err := s.music.Read()

		// Music was closed
		if err != nil {
			return
		}

		// Music has ended
		if len(samples) == 0 {
			continue
		}

		s.allEndpoints(func(endpoint *endpoint) error {
			return endpoint.streamSamples(samples)
		}, func(endpoint *endpoint, err error) {
			log.Debugf("Unable to stream samples to '%s'", endpoint.name)
		})
	}
}

func (s *Server) allEndpoints(action func(*endpoint) error, fail func(*endpoint, error)) {
	s.mutex.RLock()

	for _, endpoint := range s.endpoints {
		if err := action(endpoint); err != nil {
			fail(endpoint, err)
		}
	}

	s.mutex.RUnlock()
}

func (s *Server) handleStatusChange(endpoint *endpoint, connected bool) {
	if connected {
		log.Infof("Endpoint '%s' has connected", endpoint.name)
	} else {
		log.Infof("Endpoint '%s' has disconnected", endpoint.name)

		s.mutex.Lock()

		delete(s.endpoints, endpoint.ip.String())

		s.mutex.Unlock()
	}
}

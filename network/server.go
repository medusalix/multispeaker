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

	"github.com/medusalix/multispeaker/audio"
	"github.com/medusalix/multispeaker/log"
)

type Server struct {
	controlAddr     *net.TCPAddr
	streamAddr      *net.TCPAddr
	controlListener net.Listener
	streamListener  net.Listener
	endpoints       map[string]*Endpoint
	mutex           sync.RWMutex
	disconnect      chan *Endpoint
	music           *audio.Music
}

func NewServer(controlAddr *net.TCPAddr, streamAddr *net.TCPAddr) *Server {
	return &Server{
		controlAddr: controlAddr,
		streamAddr:  streamAddr,
		endpoints:   make(map[string]*Endpoint),
		disconnect:  make(chan *Endpoint),
		music:       audio.NewMusic(),
	}
}

func (s *Server) Run() error {
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
	go s.streamMusic()

	return nil
}

func (s *Server) GetConnectedUsers() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	users := make([]string, 0, len(s.endpoints))

	for _, endpoint := range s.endpoints {
		users = append(users, endpoint.Name)
	}

	return users
}

func (s *Server) SetVolume(user string, matchAll bool, volume int) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, endpoint := range s.endpoints {
		if !matchAll && endpoint.Name != user {
			continue
		}

		err := endpoint.ChangeVolume(volume)

		if err != nil {
			return err
		}

		// Return if single user
		if !matchAll {
			return nil
		}
	}

	return fmt.Errorf("no user with name '%s' found", user)
}

func (s *Server) PlayMusic(filePath string) error {
	if s.music.IsLoaded() {
		return errors.New("music is already playing")
	}

	sampleRate, err := s.music.Load(filePath)

	if err != nil {
		return err
	}

	err = s.preparePlayback(sampleRate)

	if err != nil {
		return err
	}

	s.music.Play()

	return nil
}

func (s *Server) StopMusic() error {
	if !s.music.IsLoaded() {
		return errors.New("music is currently not playing")
	}

	err := s.preparePlayback(0)

	if err != nil {
		return err
	}

	s.music.Stop()

	return nil
}

func (s *Server) listenControl() {
	go s.handleDisconnects()

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
			conn.Close()
			log.Debug("Duplicate connection from client with ip ", ip)
		} else {
			// New endpoint connected
			s.endpoints[ip] = NewEndpoint(conn, s.disconnect)
			log.Debug("New control connection from ", ip)
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
			if endpoint.EnableStreaming(conn) {
				log.Debug("New stream connection from ", ip)
			} else {
				// Close duplicate connection
				conn.Close()
			}
		}

		s.mutex.RUnlock()
	}
}

func (s *Server) streamMusic() {
	for {
		samples := <-s.music.Samples

		s.mutex.RLock()

		for _, endpoint := range s.endpoints {
			err := endpoint.StreamSamples(samples)

			if err != nil {
				log.Errorf("Unable to stream samples to '%s': %s", endpoint.Name, err)
			}
		}

		s.mutex.RUnlock()
	}
}

func (s *Server) preparePlayback(sampleRate int) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, endpoint := range s.endpoints {
		err := endpoint.PreparePlayback(sampleRate)

		if err != nil {
			return fmt.Errorf("unable to prepare playback for '%s': %s", endpoint.Name, err)
		}
	}

	return nil
}

func (s *Server) handleDisconnects() {
	for {
		endpoint := <-s.disconnect

		s.mutex.Lock()

		delete(s.endpoints, endpoint.Ip.String())

		s.mutex.Unlock()
	}
}

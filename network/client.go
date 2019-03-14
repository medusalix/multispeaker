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
	"os/user"
	"runtime"
	"strings"
	"time"

	volume "github.com/itchyny/volume-go"
	"github.com/medusalix/multispeaker/audio"
	"github.com/medusalix/multispeaker/log"
)

const reconnectDelay = time.Second * 5

// Client is used to connect to the server and stream music
type Client struct {
	controlAddr *net.TCPAddr
	streamAddr  *net.TCPAddr
	control     *protocol
	stream      *protocol
	player      *audio.Player
}

// NewClient constructs a new client
func NewClient(controlAddr *net.TCPAddr, streamAddr *net.TCPAddr) *Client {
	return &Client{
		controlAddr: controlAddr,
		streamAddr:  streamAddr,
		player:      audio.NewPlayer(),
	}
}

// Start starts the client
func (c *Client) Start() error {
	for {
		if err := c.run(); err != nil {
			log.Error("Connection error: ", err)
		}

		log.Info("Reconnecting")
		time.Sleep(reconnectDelay)
	}
}

func (c *Client) run() error {
	if err := c.connectControl(); err != nil {
		return err
	}

	log.Info("Connected to server")

	if err := c.announce(); err != nil {
		return err
	}

	c.listen()

	return errors.New("connection lost")
}

func (c *Client) connectControl() error {
	conn, err := net.DialTCP("tcp", nil, c.controlAddr)

	if err != nil {
		return err
	}

	c.control = newProtocol(conn)

	return nil
}

func (c *Client) connectStream() error {
	conn, err := net.DialTCP("tcp", nil, c.streamAddr)

	if err != nil {
		return err
	}

	c.stream = newProtocol(conn)

	return nil
}

func (c *Client) announce() error {
	username, err := c.getUsername()

	if err != nil {
		return err
	}

	return c.control.send(&announcePacket{
		name: username,
	})
}

func (c *Client) getUsername() (string, error) {
	currentUser, err := user.Current()

	if err != nil {
		return "", err
	}

	name := currentUser.Username

	if runtime.GOOS == "windows" {
		domainIndex := strings.IndexByte(name, '\\')

		return name[domainIndex+1:], nil
	}

	return name, nil
}

func (c *Client) listen() error {
	for {
		packet, err := c.control.receive()

		if err != nil {
			return err
		}

		switch p := packet.(type) {
		case *preparePacket:
			if err := c.preparePlayer(p.sampleRate); err != nil {
				log.Error("Error handling prepare packet: ", err)
			}
		case *volumePacket:
			if err := c.changeVolume(p.volume); err != nil {
				log.Error("Error handling volume packet: ", err)
			}
		}
	}
}

func (c *Client) preparePlayer(sampleRate int) error {
	if err := c.player.Close(); err != nil {
		// Only log error, happens sometimes
		log.Error("Error closing player: ", err)
	}

	if sampleRate == 0 {
		return nil
	}

	log.Debug("Preparing player")

	if err := c.player.Prepare(sampleRate); err != nil {
		return err
	}

	log.Debug("Connecting stream")

	if err := c.connectStream(); err != nil {
		return err
	}

	log.Info("Starting music playback")

	go c.streamMusic()

	return nil
}

func (c *Client) streamMusic() {
	for {
		samples, err := c.stream.receiveRaw()

		if err != nil {
			log.Info("Stream disconnected by server")

			break
		}

		_, err = c.player.Write(samples)

		if err != nil {
			log.Info("Music playback stopped")

			break
		}
	}
}

func (c *Client) changeVolume(vol int) error {
	// Throws error when already unmuted
	volume.Unmute()

	log.Infof("Setting volume to '%d'", vol)

	return volume.SetVolume(vol)
}

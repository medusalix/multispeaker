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
	"io"
	"net"
	"os/user"
	"runtime"
	"strings"
	"sync"
	"time"

	volume "github.com/itchyny/volume-go"
	"github.com/medusalix/multispeaker/audio"
	"github.com/medusalix/multispeaker/log"
)

const reconnectDelay = time.Second * 2

type Client struct {
	controlAddr *net.TCPAddr
	streamAddr  *net.TCPAddr
	control     *Protocol
	stream      *Protocol
	player      *audio.Player
	exit        bool
}

func NewClient(controlAddr *net.TCPAddr, streamAddr *net.TCPAddr) *Client {
	return &Client{
		controlAddr: controlAddr,
		streamAddr:  streamAddr,
		player:      audio.NewPlayer(),
	}
}

func (c *Client) Run() error {
	var group sync.WaitGroup

	for {
		log.Info("Reconnecting...")

		c.reconnect()

		group.Add(2)
		go c.listenControl(&group)
		go c.listenStream(&group)

		err := c.announce()

		if err != nil {
			return err
		}

		log.Info("Ready to play")

		group.Wait()

		log.Error("Connection to server lost")

		if c.exit {
			log.Info("Exiting...")

			break
		}

		time.Sleep(reconnectDelay)
	}

	return nil
}

func (c *Client) reconnect() {
	for {
		controlConn, err := net.DialTCP("tcp", nil, c.controlAddr)

		if err != nil {
			continue
		}

		streamConn, err := net.DialTCP("tcp", nil, c.streamAddr)

		if err != nil {
			continue
		}

		c.control = NewProtocol(controlConn)
		c.stream = NewProtocol(streamConn)

		return
	}
}

func (c *Client) listenControl(group *sync.WaitGroup) {
	for {
		packet, err := c.control.Receive()

		if err != nil {
			// Exit if closed by remote host
			if err == io.EOF {
				c.exit = true
			}

			group.Done()

			return
		}

		switch p := packet.(type) {
		case *PreparePacket:
			err := c.updatePlayer(int(p.SampleRate))

			if err != nil {
				log.Error("Error handling prepare packet: ", err)

				continue
			}
		case *ControlPacket:
			err := c.updateVolume(int(p.Volume))

			if err != nil {
				log.Error("Error handling control packet: ", err)
			}
		}
	}
}

func (c *Client) listenStream(group *sync.WaitGroup) {
	for {
		samples, err := c.stream.ReceiveRaw()

		if err != nil {
			group.Done()

			return
		}

		c.player.Write(samples)
	}
}

func (c *Client) announce() error {
	username, err := getUsername()

	if err != nil {
		return err
	}

	return c.control.Send(&AnnouncePacket{
		Name: []byte(username),
	})
}

func getUsername() (string, error) {
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

func (c *Client) updatePlayer(sampleRate int) error {
	c.player.Stop()

	if sampleRate > 0 {
		log.Info("Preparing player with sample rate ", sampleRate)

		return c.player.Prepare(sampleRate)
	}

	log.Info("Stopping player")

	return nil
}

func (c *Client) updateVolume(vol int) error {
	// Checking volume.GetMuted leads to a crash
	// So we ignore errors due to not being muted
	volume.Unmute()

	log.Info("Setting volume to ", vol)

	return volume.SetVolume(vol)
}

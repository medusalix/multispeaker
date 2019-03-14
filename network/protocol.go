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
)

const sendBufferSize = 1024
const receiveBufferSize = 1024

const (
	announcePacketID = iota
	preparePacketID
	controlPacketID
)

type protocol struct {
	conn          net.Conn
	sendBuffer    []byte
	receiveBuffer []byte
}

type packet interface {
	encode(buffer []byte)
	decode(buffer []byte)
	size() int
}

// ................
// Client -> Server
// ................

// announcePacket notifies the server of a new client
type announcePacket struct {
	// Name of the client (username)
	name string
}

// ................
// Server -> Client
// ................

// preparePacket creates/closes the client's player
type preparePacket struct {
	// Sample rate for music playback
	// > 0 -> Create new player
	// = 0 -> Close player
	sampleRate int
}

// volumePacket sets the client's volume
type volumePacket struct {
	// Volume for the operating system
	// In range 0 <= volume <= 100
	volume int
}

func newProtocol(conn net.Conn) *protocol {
	return &protocol{
		conn:          conn,
		sendBuffer:    make([]byte, sendBufferSize),
		receiveBuffer: make([]byte, receiveBufferSize),
	}
}

func (p *protocol) send(packet packet) error {
	var packetID int

	switch packet.(type) {
	case *announcePacket:
		packetID = announcePacketID
	case *preparePacket:
		packetID = preparePacketID
	case *volumePacket:
		packetID = controlPacketID
	default:
		return errors.New("unable to transmit packet with unknown id")
	}

	size := packet.size()

	p.sendBuffer[0] = byte(packetID)
	p.sendBuffer[1] = byte(size >> 8)
	p.sendBuffer[2] = byte(size)

	packet.encode(p.sendBuffer[3:])

	_, err := p.conn.Write(p.sendBuffer[:3+size])

	return err
}

func (p *protocol) sendRaw(data []byte) (int, error) {
	return p.conn.Write(data)
}

func (p *protocol) receive() (packet, error) {
	_, err := p.conn.Read(p.receiveBuffer[:3])

	if err != nil {
		return nil, err
	}

	var packet packet

	switch p.receiveBuffer[0] {
	case announcePacketID:
		packet = &announcePacket{}
	case preparePacketID:
		packet = &preparePacket{}
	case controlPacketID:
		packet = &volumePacket{}
	default:
		return nil, errors.New("received packet with unknown id")
	}

	// Decode size from packet header
	size := int(p.receiveBuffer[1])<<8 | int(p.receiveBuffer[2])

	if size < packet.size() {
		return nil, errors.New("received invalid packet")
	}

	received := 0

	// Restore fragmented packets
	for received < size {
		n, err := p.conn.Read(p.receiveBuffer[received:size])

		if err != nil {
			return nil, err
		}

		received += n
	}

	packet.decode(p.receiveBuffer[:size])

	return packet, nil
}

func (p *protocol) receiveRaw() ([]byte, error) {
	n, err := p.conn.Read(p.receiveBuffer)

	return p.receiveBuffer[:n], err
}

func (p *protocol) close() error {
	return p.conn.Close()
}

func (p *announcePacket) encode(buffer []byte) {
	copy(buffer, p.name)
}

func (p *preparePacket) encode(buffer []byte) {
	buffer[0] = byte(p.sampleRate >> 8)
	buffer[1] = byte(p.sampleRate)
}

func (p *volumePacket) encode(buffer []byte) {
	buffer[0] = byte(p.volume)
}

func (p *announcePacket) decode(buffer []byte) {
	p.name = string(buffer)
}

func (p *preparePacket) decode(buffer []byte) {
	p.sampleRate = int(buffer[0])<<8 | int(buffer[1])
}

func (p *volumePacket) decode(buffer []byte) {
	p.volume = int(buffer[0])
}

func (p *announcePacket) size() int {
	return len(p.name)
}

func (p *preparePacket) size() int {
	return 2
}

func (p *volumePacket) size() int {
	return 1
}

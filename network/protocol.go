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
	announcePacketId = iota
	preparePacketId
	controlPacketId
)

type Protocol struct {
	conn          net.Conn
	sendBuffer    []byte
	receiveBuffer []byte
}

type Packet interface {
	encode(buffer []byte)
	decode(buffer []byte)
	size() int
}

type AnnouncePacket struct {
	// Name of the client (username)
	Name []uint8
}

type PreparePacket struct {
	// Sample rate for music playback
	// > 0 -> Create new player
	// = 0 -> Close player
	SampleRate uint16
}

type ControlPacket struct {
	// Volume for the operating system
	// In range 0 <= volume <= 100
	Volume uint8
}

func NewProtocol(conn net.Conn) *Protocol {
	return &Protocol{
		conn:          conn,
		sendBuffer:    make([]byte, sendBufferSize),
		receiveBuffer: make([]byte, receiveBufferSize),
	}
}

func (p *Protocol) Send(packet Packet) error {
	var packetId uint8

	switch packet.(type) {
	case *AnnouncePacket:
		packetId = announcePacketId
	case *PreparePacket:
		packetId = preparePacketId
	case *ControlPacket:
		packetId = controlPacketId
	default:
		return errors.New("unable to transmit packet with unknown id")
	}

	size := packet.size()

	p.sendBuffer[0] = packetId
	p.sendBuffer[1] = byte(size >> 8)
	p.sendBuffer[2] = byte(size)

	packet.encode(p.sendBuffer[3:])

	_, err := p.conn.Write(p.sendBuffer[:3+size])

	return err
}

func (p *Protocol) SendRaw(data []byte) (int, error) {
	return p.conn.Write(data)
}

func (p *Protocol) Receive() (Packet, error) {
	_, err := p.conn.Read(p.receiveBuffer[:3])

	if err != nil {
		return nil, err
	}

	var packet Packet

	switch p.receiveBuffer[0] {
	case announcePacketId:
		packet = &AnnouncePacket{}
	case preparePacketId:
		packet = &PreparePacket{}
	case controlPacketId:
		packet = &ControlPacket{}
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

func (p *Protocol) ReceiveRaw() ([]byte, error) {
	n, err := p.conn.Read(p.receiveBuffer)

	return p.receiveBuffer[:n], err
}

func (p *AnnouncePacket) encode(buffer []byte) {
	copy(buffer, p.Name)
}

func (p *PreparePacket) encode(buffer []byte) {
	buffer[0] = byte(p.SampleRate >> 8)
	buffer[1] = byte(p.SampleRate)
}

func (p *ControlPacket) encode(buffer []byte) {
	buffer[0] = p.Volume
}

func (p *AnnouncePacket) decode(buffer []byte) {
	p.Name = buffer
}

func (p *PreparePacket) decode(buffer []byte) {
	p.SampleRate = uint16(buffer[0])<<8 | uint16(buffer[1])
}

func (p *ControlPacket) decode(buffer []byte) {
	p.Volume = buffer[0]
}

func (p *AnnouncePacket) size() int {
	return len(p.Name)
}

func (p *PreparePacket) size() int {
	return 2
}

func (p *ControlPacket) size() int {
	return 1
}

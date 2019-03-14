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

package audio

import (
	"github.com/hajimehoshi/oto"
)

const playerBufferSize = 8192

// Player is used to play music from samples
type Player struct {
	context *oto.Context
	player  *oto.Player
}

// NewPlayer constructs a new music player
func NewPlayer() *Player {
	return &Player{}
}

// Prepare sets the sample rate of the player
func (p *Player) Prepare(sampleRate int) error {
	var err error

	// Context with 16 bit PCM, 2 channels
	p.context, err = oto.NewContext(sampleRate, 2, 2, playerBufferSize)

	if err != nil {
		return err
	}

	p.player = p.context.NewPlayer()

	return nil
}

// Write writes the given samples to the player
func (p *Player) Write(samples []byte) (int, error) {
	if p.player == nil {
		return 0, nil
	}

	return p.player.Write(samples)
}

// Close closes the player
func (p *Player) Close() error {
	if p.player == nil {
		return nil
	}

	// Ignore errors, close context anyways
	p.player.Close()

	return p.context.Close()
}

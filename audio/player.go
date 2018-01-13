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

type Player struct {
	pcmPlayer *oto.Player
}

func NewPlayer() *Player {
	return &Player{}
}

func (p *Player) Prepare(sampleRate int) error {
	var err error

	// Player with 16 bit PCM, 2 channels
	p.pcmPlayer, err = oto.NewPlayer(sampleRate, 2, 2, playerBufferSize)

	return err
}

func (p *Player) Write(samples []byte) {
	if p.pcmPlayer != nil {
		p.pcmPlayer.Write(samples)
	}
}

func (p *Player) Stop() {
	if p.pcmPlayer != nil {
		p.pcmPlayer.Close()
		p.pcmPlayer = nil
	}
}

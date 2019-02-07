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
	"io"
	"os"

	mp3 "github.com/hajimehoshi/go-mp3"
	"github.com/medusalix/multispeaker/log"
)

const musicBufferSize = 512

type Music struct {
	Samples chan []byte
	decoder *mp3.Decoder
	stop    chan bool
}

func NewMusic() *Music {
	return &Music{
		Samples: make(chan []byte),
		stop:    make(chan bool),
	}
}

func (m *Music) Load(filePath string) (int, error) {
	file, err := os.Open(filePath)

	if err != nil {
		return 0, err
	}

	m.decoder, err = mp3.NewDecoder(file)

	if err != nil {
		return 0, err
	}

	return m.decoder.SampleRate(), nil
}

func (m *Music) IsLoaded() bool {
	return m.decoder != nil
}

func (m *Music) Play() {
	go m.readSamples()
}

func (m *Music) Stop() {
	m.stop <- true
}

func (m *Music) readSamples() {
	for {
		select {
		case <-m.stop:
			m.decoder.Close()
			m.decoder = nil

			return
		default:
			// Samples are 16 bit, 2 channels
			samples := make([]byte, musicBufferSize)
			frame := make([]byte, 4)

			for i := 0; i < len(samples); i += 4 {
				n, err := m.decoder.Read(frame)

				if err != nil {
					if err == io.EOF {
						m.decoder.Close()
						m.decoder = nil

						return
					} else {
						log.Error("Error reading music: ", err)
					}
				}

				copy(samples[i:], frame[:n])
			}

			m.Samples <- samples
		}
	}
}

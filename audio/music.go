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
)

const musicBufferSize = 512

// Music is used to read samples from a music file
type Music struct {
	decoder *mp3.Decoder
}

// NewMusic constructs a new music reader
func NewMusic() *Music {
	return &Music{}
}

// Load loads a music file from a given path
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

// Read reads samples from the music file
func (m *Music) Read() ([]byte, error) {
	// Samples are 16 bit, 2 channels
	samples := make([]byte, musicBufferSize)
	frame := make([]byte, 4)

	for i := 0; i < len(samples); i += 4 {
		n, err := m.decoder.Read(frame)

		if err != nil {
			// Music reached the end
			if err == io.EOF {
				return samples[:i], nil
			}

			return nil, err
		}

		copy(samples[i:], frame[:n])
	}

	return samples, nil
}

// Close closes the music
func (m *Music) Close() error {
	return m.decoder.Close()
}

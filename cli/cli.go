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

package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/medusalix/multispeaker/network"
)

// Prompt specifies the prefix before each prompt
var Prompt string

var mutex sync.Mutex
var commands = map[string]func(*network.Server, ...string){
	"help": help,
	"list": listUsers,
	"play": playMusic,
	"stop": stopMusic,
	"vol":  changeVolume,
}

// Writeln writes to standard output with a newline
func Writeln(params ...interface{}) {
	mutex.Lock()
	fmt.Print("\r")
	fmt.Println(params...)
	fmt.Print(Prompt)
	mutex.Unlock()
}

// Writef formats the parameters and writes to standard output
func Writef(format string, params ...interface{}) {
	Writeln(fmt.Sprintf(format, params...))
}

// HandleCommands reads from standard input and handles the commands
func HandleCommands(server *network.Server) {
	reader := bufio.NewReader(os.Stdin)

	for {
		mutex.Lock()
		fmt.Print("\r" + Prompt)
		mutex.Unlock()

		input, err := reader.ReadString('\n')

		if err != nil {
			// Exit when Ctrl-C is pressed
			if err == io.EOF {
				break
			}

			Writeln("Error while reading command:", err)

			continue
		}

		parts := parseInput(input)
		commandName := strings.ToLower(parts[0])
		args := parts[1:]

		if commandName == "exit" {
			break
		}

		command, ok := commands[commandName]

		if !ok {
			Writeln("Unknown command")

			continue
		}

		command(server, args...)
	}
}

func help(server *network.Server, args ...string) {
	Writeln(
		"Commands:\n\n" +
			"list: Prints a list of all currently connected users.\n" +
			"play <file>: Starts playback of a specified MP3 file.\n" +
			"stop: Stops the music playback.\n" +
			"vol <user|all> <volume>: Sets the system volume of a users's computer.\n" +
			"If all is supplied, the volume of all connected users is changed.\n" +
			"exit: Exits the program.",
	)
}

func listUsers(server *network.Server, args ...string) {
	for _, user := range server.GetConnectedUsers() {
		Writeln(user)
	}
}

func playMusic(server *network.Server, args ...string) {
	if len(args) < 1 {
		Writeln("Args: <file>")

		return
	}

	if err := server.PlayMusic(args[0]); err != nil {
		Writeln("Error starting music playback:", err)
	} else {
		Writeln("Started music playback")
	}
}

func stopMusic(server *network.Server, args ...string) {
	if err := server.StopMusic(); err != nil {
		Writeln("Error stopping music playback:", err)
	} else {
		Writeln("Stopped music playback")
	}
}

func changeVolume(server *network.Server, args ...string) {
	if len(args) < 2 {
		Writeln("Args: <user|all> <volume>")

		return
	}

	user := args[0]
	volume, err := strconv.Atoi(args[1])

	if err != nil {
		Writeln("Invalid volume:", err)
	} else if volume < 0 {
		Writeln("Volume can't be smaller than 0")
	} else if volume > 100 {
		Writeln("Volume can't be greater than 100")
	} else if err := server.SetVolume(user, volume); err != nil {
		Writeln("Error setting volume:", err)
	} else {
		Writef("Set volume of user '%s' to '%d'", user, volume)
	}
}

func parseInput(input string) []string {
	input = strings.TrimSuffix(input, "\n")
	input = strings.TrimSuffix(input, "\r")

	output := make([]string, 0)
	quotes := false

	lastSplit := 0

	for i, char := range input {
		if char == '"' {
			quotes = !quotes
		} else if !quotes && char == ' ' {
			part := input[lastSplit:i]
			output = append(output, strings.Trim(part, "\""))
			lastSplit = i + 1
		}
	}

	part := input[lastSplit:]
	output = append(output, strings.Trim(part, "\""))

	return output
}

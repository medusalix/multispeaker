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
	"runtime"
	"syscall"
	"unsafe"
)

func HideConsole() {
	if runtime.GOOS != "windows" {
		return
	}

	kernel := syscall.NewLazyDLL("kernel32.dll")
	user := syscall.NewLazyDLL("user32.dll")

	getConsoleWindow := kernel.NewProc("GetConsoleWindow")
	showWindow := user.NewProc("ShowWindow")
	messageBox := user.NewProc("MessageBoxW")

	// Call ShowWindow with SW_HIDE (0)
	window, _, _ := getConsoleWindow.Call()
	showWindow.Call(window, 0)

	text, _ := syscall.UTF16PtrFromString("Multispeaker is now running in the background!")
	caption, _ := syscall.UTF16PtrFromString("multispeaker")

	// Call MessageBoxW with MB_ICONINFORMATION (0x40)
	messageBox.Call(
		0,
		uintptr(unsafe.Pointer(text)),
		uintptr(unsafe.Pointer(caption)),
		0x40,
	)
}

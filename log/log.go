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

package log

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	debug = iota
	info
	err
)

var levels = []string{
	"DEBUG",
	"INFO",
	"ERROR",
}

var logWriter func(format string, params ...interface{})
var logLevel int

func Init(writer func(format string, params ...interface{}), level string) {
	logWriter = writer

	for i, levelName := range levels {
		if strings.EqualFold(level, levelName) {
			logLevel = i

			break
		}
	}
}

func Debug(params ...interface{}) {
	log(debug, fmt.Sprint(params...))
}

func Debugf(format string, params ...interface{}) {
	log(debug, fmt.Sprintf(format, params...))
}

func Info(params ...interface{}) {
	log(info, fmt.Sprint(params...))
}

func Infof(format string, params ...interface{}) {
	log(info, fmt.Sprintf(format, params...))
}

func Error(params ...interface{}) {
	log(err, fmt.Sprint(params...))
}

func Errorf(format string, params ...interface{}) {
	log(err, fmt.Sprintf(format, params...))
}

func log(level int, message string) {
	if level < logLevel {
		return
	}

	date := time.Now().Format("02.01.2006 15:04:05")
	levelText := levels[level]
	_, path, _, _ := runtime.Caller(2)
	dir, file := filepath.Split(path)

	// Combine last dir and filename without extension
	dir = filepath.Base(dir)
	file = strings.TrimSuffix(file, filepath.Ext(file))
	fullPath := dir + "/" + file

	logWriter("%s %-5s %s - %s\n", date, levelText, fullPath, message)
}

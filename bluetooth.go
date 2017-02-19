package main

import (
	"bufio"
	"bytes"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

func indentDepth(line []byte) int {
	i := 0
	for {
		if bytes.HasPrefix(line, []byte("    ")) {
			i++
			line = bytes.TrimPrefix(line, []byte("    "))
		} else {
			return i
		}
	}
}

var durationExpr = regexp.MustCompile(`(\d+)\s+(\w+)`)

func bluetoothData() (map[string]interface{}, error) {
	cmd := exec.Command("system_profiler", "SPBluetoothDataType")
	var outb bytes.Buffer
	cmd.Stdout = &outb
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return parseBluetoothData(&outb)
}

func bluetoothConnected(device string) (bool, error) {
	data, err := bluetoothData()
	if err != nil {
		return false, err
	}
	if bluetooth, ok := data["Bluetooth"].(map[string]interface{}); ok {
		if devices, ok := bluetooth["Devices (Paired, Configured, etc.)"].(map[string]interface{}); ok {
			if device, ok := devices[device].(map[string]interface{}); ok {
				if connected, ok := device["Connected"].(bool); ok {
					return connected, nil
				}
			}
		}
	}
	return false, nil
}

func parseBluetoothData(r io.Reader) (map[string]interface{}, error) {
	data := map[string]interface{}{}
	currentData := map[int]map[string]interface{}{0: data}

	bufferedReader := bufio.NewReader(r)
	for {
		lb, _, err := bufferedReader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Skip empty lines
		if len(bytes.TrimSpace(lb)) == 0 {
			continue
		}

		depth := indentDepth(lb)
		lbt := bytes.TrimSpace(lb)
		split := bytes.Split(lbt, []byte(":"))
		split[1] = bytes.TrimSpace(split[1])

		if len(split[1]) == 0 {
			// Heading :\n == section heading, increase depth
			currentData[depth][string(split[0])] = map[string]interface{}{}
			currentData[depth+1] = currentData[depth][string(split[0])].(map[string]interface{})
			continue
		}

		// A bunch of boolean conversions
		if bytes.EqualFold(split[1], []byte("Yes")) || bytes.EqualFold(split[1], []byte("On")) || bytes.EqualFold(split[1], []byte("Enabled")) || bytes.EqualFold(split[1], []byte("No")) || bytes.EqualFold(split[1], []byte("Off")) || bytes.EqualFold(split[1], []byte("Disabled")) {
			currentData[depth][string(split[0])] = bytes.EqualFold(split[1], []byte("Yes")) || bytes.EqualFold(split[1], []byte("On")) || bytes.EqualFold(split[1], []byte("Enabled"))
			continue
		}

		// Duration conversion
		if op := durationExpr.ReplaceAll(split[1], []byte("$1$2")); len(op) > 0 {
			if d, err := time.ParseDuration(string(op)); err == nil {
				currentData[depth][string(split[0])] = d
				continue
			}
		}

		// Int conversion
		if i, err := strconv.Atoi(string(split[1])); err == nil {
			currentData[depth][string(split[0])] = i
			continue
		}

		currentData[depth][string(split[0])] = string(split[1])
	}

	return data, nil
}

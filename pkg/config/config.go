/*
Copyright © 2025 Ian Shuley

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// ReadConfigFile reads and unmarshals a JSON config file
func ReadConfigFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// WriteConfigFileSecurely marshals and writes any config data to JSON with secure permissions
func WriteConfigFileSecurely(path string, data any, atomicWriteFunc func(string, []byte, os.FileMode) error) error {
	encoded, err := MarshalConfigData(data)
	if err != nil {
		return err
	}
	return atomicWriteFunc(path, encoded, 0600)
}

// MarshalConfigData converts config data to formatted JSON
func MarshalConfigData(data any) ([]byte, error) {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal config data: %w", err)
	}
	return encoded, nil
}

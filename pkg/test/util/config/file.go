// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"io/ioutil"
	"os"
	"strings"
)

// WriteYAML writes the given YAML content to ISTIOCONFIG pointed YAML file.
func WriteYAML(overwrite bool, contents ...string) error {
	filepath := os.Getenv("ISTIOCONFIG")
	if filepath == "" {
		filepath = os.Getenv("HOME") + string(os.PathSeparator) + ".istioctl" + string(os.PathSeparator) + "config.yaml"
	}
	if !overwrite {
		if _, err := os.Stat(filepath); err == nil {
			// file already exists and do not want to overwrite it, simply return
			return nil
		}
	}
	content := strings.Join(contents, "\n")
	err := ioutil.WriteFile(filepath, []byte(content), 0755)
	if err != nil {
		return err
	}
	return nil
}

// DeleteYAML delete the ISTIOCONFIG pointed YAML file.
func DeleteYAML() error {
	filepath := os.Getenv("ISTIOCONFIG")
	if filepath == "" {
		filepath = os.Getenv("HOME") + "/.istioctl/config.yaml"
	}
	err := os.Remove(filepath)
	if err != nil {
		return err
	}
	return nil
}

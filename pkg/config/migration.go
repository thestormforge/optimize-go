/*
Copyright 2020 GramLabs, Inc.

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
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
)

// legacyConfig contains only the parts of the legacy configuration object that we can migrate; the address and
// credentials were all invalidated when the remote server switched to a single endpoint
type legacyConfig struct {
	Manager legacyManager `json:"manager"`
}

// legacyManager shares the current representation to make migration easier
type legacyManager struct {
	Environment []ControllerEnvVar `json:"env"`
}

// migrationLoader will take the meaningful bits from a legacy config file and delete that file once the changes are persisted
func migrationLoader(cfg *RedSkyConfig) error {
	// Try to rename the old file to the new location
	old, _ := configFilename("redsky/config")
	if err := safeRenameConfig(old, cfg.Filename); err == nil {
		// It is safe to merge in the contents if the environment loader ONLY does defaults
		return fileLoader(cfg)
	}

	// This is _really_ legacy at this point, we may want to consider dropping support
	filename := filepath.Join(os.Getenv("HOME"), ".redsky")
	name := "default"

	// Use the current cluster name as the default name for controller
	cmd := exec.Command("kubectl", "config", "view", "--minify", "--output", "jsonpath={.clusters[0].name}")
	if stdout, err := cmd.Output(); err == nil {
		name = strings.TrimSpace(string(stdout))
	}

	lc, err := loadLegacyConfigFile(filename)
	if err != nil {
		return err
	}

	if lc == nil || len(lc.Manager.Environment) == 0 {
		return nil
	}

	apply := func(cfg *Config) {
		mergeControllers(cfg, []NamedController{{Name: name, Controller: Controller{Env: lc.Manager.Environment}}})
	}

	// We can't use cfg.Update here because we only want to remove the file as part of cfg.Write
	apply(&cfg.data)
	cfg.unpersisted = append(cfg.unpersisted, func(cfg *Config) error {
		apply(cfg)
		return os.Remove(filename)
	})

	return nil
}

// safeRenameConfig delegates to `os.Rename`, but only if the destination file does not exist.
func safeRenameConfig(oldname, newname string) error {
	// TODO Should we check if parent directories are the same and just rename the directories?

	// Fail if the destination already exists
	if _, err := os.Lstat(newname); err == nil {
		return &os.LinkError{Op: "rename", Old: oldname, New: newname, Err: os.ErrExist}
	}

	// Only create the directory for the new path if the old path exists (otherwise, let os.Rename fail)
	if _, err := os.Lstat(oldname); err == nil {
		if err := os.MkdirAll(filepath.Dir(newname), 0700); err != nil {
			return err
		}
	}

	if err := os.Rename(oldname, newname); err != nil {
		return err
	}

	_ = os.Remove(filepath.Dir(oldname))
	return nil
}

// loadLegacyConfigFile will read the specified file into the legacy configuration
func loadLegacyConfigFile(filename string) (*legacyConfig, error) {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lc := &legacyConfig{}
	if err = yaml2.NewYAMLOrJSONDecoder(bufio.NewReader(f), 4096).Decode(lc); err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	return lc, nil
}

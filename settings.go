package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

var (
	settingPath string
)

type settings struct {
	ActiveTenant string `json:"activeTenant"`
}

func defaultSettingsPath() string {
	if settingPath != "" {
		return settingPath
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%s/.armclient/settings.json", usr.HomeDir)
}

func setDefaultSettingsPath(path string) {
	settingPath = path
}

func saveSettings(setting settings) error {
	path := defaultSettingsPath()
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	newFile, err := ioutil.TempFile(dir, "tmp")
	if err != nil {
		return fmt.Errorf("failed to create the temp file: %v", err)
	}
	tempPath := newFile.Name()

	if err := json.NewEncoder(newFile).Encode(setting); err != nil {
		return fmt.Errorf("failed to encode to file %s: %v", tempPath, err)
	}
	if err := newFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file %s: %v", tempPath, err)
	}

	// Atomic replace to avoid multi-writer file corruptions
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to move temporary file to desired output location. src=%s dst=%s: %v", tempPath, path, err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to chmod the file %s: %v", path, err)
	}
	return nil
}

func readSettings() (setting settings, err error) {
	path := defaultSettingsPath()
	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			return settings{}, fmt.Errorf("failed to open file %s: %v", path, err)
		}
		defer file.Close()

		dec := json.NewDecoder(file)
		if err = dec.Decode(&setting); err != nil {
			return settings{}, fmt.Errorf("failed to decode contents of file %s: %v", path, err)
		}
		return setting, nil
	}

	return settings{}, nil
}

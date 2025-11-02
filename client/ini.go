package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// INI file reader/writer
type INIFile struct {
	path     string
	sections map[string]map[string]string
}

func LoadINI(path string) (*INIFile, error) {
	ini := &INIFile{
		path:     path,
		sections: make(map[string]map[string]string),
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty file
			return ini, nil
		}
		return nil, err
	}
	defer file.Close()

	var currentSection string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			if ini.sections[currentSection] == nil {
				ini.sections[currentSection] = make(map[string]string)
			}
			continue
		}

		// Key=Value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && currentSection != "" {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			ini.sections[currentSection][key] = value
		}
	}

	return ini, scanner.Err()
}

func (ini *INIFile) Get(section, key, defaultValue string) string {
	if sec, ok := ini.sections[section]; ok {
		if val, ok := sec[key]; ok {
			return val
		}
	}
	return defaultValue
}

func (ini *INIFile) Set(section, key, value string) {
	if ini.sections[section] == nil {
		ini.sections[section] = make(map[string]string)
	}
	ini.sections[section][key] = value
}

func (ini *INIFile) Save() error {
	file, err := os.Create(ini.path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for section, keys := range ini.sections {
		fmt.Fprintf(writer, "[%s]\n", section)
		for key, value := range keys {
			fmt.Fprintf(writer, "%s=%s\n", key, value)
		}
		fmt.Fprintln(writer)
	}

	return nil
}

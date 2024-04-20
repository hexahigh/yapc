package config

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
)

type Config struct {
	Endpoint string
}

// GetString retrieves a specific key from the configuration file located at the given path.
//
// Parameters:
// - path: a string representing the path to the configuration file.
// - key: a string representing the key to retrieve from the configuration.
// Returns a string value corresponding to the requested key or an empty string if the key is not found.
func GetString(path string, key string) string {
	config, _ := ReadFromFile(path)
	switch key {
	case "Endpoint":
		return config.Endpoint
	default:
		return ""
	}
}

// ReadFromFile reads and parses a configuration file from the given path.
//
// Parameters:
// - path: a string representing the path to the configuration file.
// Returns a Config struct containing the configuration data and an error if any.
func ReadFromFile(path string) (Config, error) {
	var config Config
	file, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

// GetDefaultLocation returns the default location of the configuration file.
//
// It returns a string representing the default location of the configuration file.
func GetDefaultLocation() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}

	location := path.Join(configDir, "yapc-cli", "config.json")

	return location
}

// GenerateDefaultConfig generates a default configuration file at the specified path.
//
// Parameters:
// - path: a string representing the path where the configuration file should be created.
// Returns an error if any.
func GenerateDefaultConfig(path string) error {
	config := Config{
		Endpoint: "https://pomf1.080609.xyz",
	}

	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	err = json.NewEncoder(file).Encode(config)
	if err != nil {
		return err
	}
	return nil
}

// CheckAndGenerate checks if the specified file exists at the given path. If the file does not exist, it generates a default configuration file at the path using the GenerateDefaultConfig function.
//
// Parameters:
// - path: a string representing the path to the file.
//
// Returns:
// - error: an error if there was a problem checking the file existence or generating the default configuration file.
func CheckAndGenerate(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return GenerateDefaultConfig(path)
	}
	return nil
}

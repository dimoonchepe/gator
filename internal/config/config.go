package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Config struct {
	DBURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() Config {
	// Read JSON file found at ~/.gatorconfig.json
	file, err := os.Open(os.Getenv("HOME") + "/.gatorconfig.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return Config{}
	}
	defer file.Close()

	// Read JSON data from file
	var config Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return Config{}
	}
	return config
}

func (c Config) SetUser(user string) error {
	// write user to existing config file
	file, err := os.OpenFile(os.Getenv("HOME")+"/.gatorconfig.json", os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return errors.New("error opening file")
	}
	defer file.Close()

	// write JSON data to file
	updatedConfig := Config{
		DBURL:           c.DBURL,
		CurrentUserName: user,
	}
	err = json.NewEncoder(file).Encode(updatedConfig)
	if err != nil {
		return fmt.Errorf("error encoding JSON: %v", err)
	}

	return nil
}

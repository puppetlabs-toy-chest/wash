package analytics

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

// Config represents Wash's Google Analytics Config
type Config struct {
	Disabled bool      `yaml:"disabled"`
	UserID   uuid.UUID `yaml:"user-id"`
}

// GetConfig returns Wash's analytics config, which is located at
// $HOME/.puppetlabs/wash/analytics.yaml.
func GetConfig() (config Config, err error) {
	// Check if analytics is disabled via the environment before
	// returning
	defer func() {
		if err != nil {
			return
		}
		disabledStr, ok := os.LookupEnv("WASH_DISABLE_ANALYTICS")
		if !ok {
			return
		}
		// Doing disabledBool, err := ... will not write to the err return
		// value. It will instead create a new err variable within the defer
		// func's scope. To avoid this, declare disabledBool so that we can
		// use "=" instead of ":="
		var disabledBool bool
		disabledBool, err = strconv.ParseBool(disabledStr)
		if err != nil {
			err = fmt.Errorf("WASH_DISABLE_ANALYTICS is set to %v. Valid values are 'true' or 'false'", disabledStr)
			return
		}
		config.Disabled = disabledBool
	}()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	analyticsConfigFile := filepath.Join(homeDir, ".puppetlabs", "wash", "analytics.yaml")
	config, err = readAnalyticsConfigFile(analyticsConfigFile)
	if err == nil || !os.IsNotExist(err) {
		return
	}
	// The analyticsConfigFile does not exist, so create one. First, read
	// Bolt's analytics config file (if it exists) so we can re-use its
	// user ID
	config, err = readAnalyticsConfigFile(
		filepath.Join(homeDir, ".puppetlabs", "bolt", "analytics.yaml"),
	)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if config.UserID == uuid.Nil {
		config.UserID = uuid.New()
	}
	config.Disabled = false
	bytes, err := yaml.Marshal(config)
	if err != nil {
		// This should never happen
		err = fmt.Errorf("could not marshal the analytics config: %v", err)
		return
	}
	err = ioutil.WriteFile(analyticsConfigFile, bytes, 0644)
	return
}

func readAnalyticsConfigFile(path string) (Config, error) {
	_, err := os.Stat(path)
	if err != nil {
		return Config{}, err
	}
	rawConfig, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("error reading %v: %v", path, err)
	}
	var config Config
	if err := yaml.Unmarshal(rawConfig, &config); err != nil {
		return config, fmt.Errorf(
			"error reading %v: could not unmarshal the analytics config: %v",
			path,
			err,
		)
	}
	return config, nil
}

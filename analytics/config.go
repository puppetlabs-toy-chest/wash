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
func GetConfig() (Config, error) {
	config := Config{}

	// Check if analytics is disabled via the environment
	if disabledStr, ok := os.LookupEnv("WASH_DISABLE_ANALYTICS"); ok {
		disabledBool, err := strconv.ParseBool(disabledStr)
		if err != nil {
			return config, fmt.Errorf("WASH_DISABLE_ANALYTICS is set to %v. Valid values are 'true' or 'false'", disabledStr)
		}
		config.Disabled = disabledBool
		if config.Disabled {
			// Analytics is disabled so we can return. There's no need to do anything
			// else.
			return config, nil
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config, err
	}
	analyticsConfigFile := filepath.Join(homeDir, ".puppetlabs", "wash", "analytics.yaml")
	config, err = readAnalyticsConfigFile(analyticsConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return config, newConfigReadErr(analyticsConfigFile, err)
	}
	if config.Disabled || config.UserID != uuid.Nil {
		return config, nil
	}
	// Analytics is not disabled, but no user ID is set. Thus, we'll need
	// to generate a user ID and then write that user ID to the analytics
	// config file. Note that this also handles the case of a nonexistant
	// config file.
	//
	// First, try using Bolt's user ID if it's available.
	boltAnalyticsConfigFile := filepath.Join(homeDir, ".puppetlabs", "bolt", "analytics.yaml")
	boltAnalyticsConfig, err := readAnalyticsConfigFile(boltAnalyticsConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return config, newConfigReadErr(boltAnalyticsConfigFile, err)
	}
	if boltAnalyticsConfig.UserID != uuid.Nil {
		config.UserID = boltAnalyticsConfig.UserID
	} else {
		config.UserID = uuid.New()
	}
	// Now write the config
	bytes, err := yaml.Marshal(config)
	if err != nil {
		// This should never happen
		return config, fmt.Errorf("could not marshal the analytics config: %v", err)
	}
	// Make sure that the ~/.puppetlabs/wash directory exists. Otherwise, ioutil.WriteFile
	// will return an error.
	if err := os.MkdirAll(filepath.Dir(analyticsConfigFile), 0750); err != nil {
		return config, err
	}
	return config, ioutil.WriteFile(analyticsConfigFile, bytes, 0644)
}

func readAnalyticsConfigFile(path string) (Config, error) {
	rawConfig, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := yaml.Unmarshal(rawConfig, &config); err != nil {
		return config, fmt.Errorf("could not unmarshal the analytics config: %v", err)
	}
	return config, nil
}

func newConfigReadErr(file string, reason error) error {
	return fmt.Errorf("error reading %v: %v", file, reason)
}

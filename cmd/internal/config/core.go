// Package config implements configuration for the wash executable using
// https://github.com/spf13/viper.
package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Contains all the keys for Wash's shared config
const (
	SocketKey   = "socket"
	EmbeddedKey = "embedded"
)

// Socket is the path to the Wash server's UNIX
// socket
var Socket string
var Embedded bool

// Init initializes the config package. It loads Wash's defaults and
// sets up viper
func Init() error {
	// Set any defaults
	cdir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	viper.SetDefault(SocketKey, filepath.Join(cdir, "wash", "wash-api.sock"))
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	defaultFileAbs = filepath.Join(homeDir, defaultFileSuffix)

	// Tell viper that the config. can be read from WASH_<entry>
	// environment variables
	viper.SetEnvPrefix("WASH")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set the config type
	viper.SetConfigType("yaml")

	// Load the shared config
	Socket = viper.GetString(SocketKey)
	Embedded = viper.GetBool(EmbeddedKey)

	return nil
}

var defaultFileSuffix = filepath.Join(".puppetlabs", "wash", "wash.yaml")
var defaultFileRel = filepath.Join("~", defaultFileSuffix)
var defaultFileAbs string

// DefaultFile returns the default config file's path
func DefaultFile() string {
	return defaultFileRel
}

// ReadFrom reads the config from the specified file.
// If file == DefaultFile(), then ReadFrom wil not return
// an error if file does not exist.
func ReadFrom(file string) error {
	if file == DefaultFile() {
		if defaultFileAbs == "" {
			panic("config.ReadFrom: default file not set. Please call config.Init()")
		}
		if _, err := os.Stat(defaultFileAbs); os.IsNotExist(err) {
			return nil
		}
		file = defaultFileAbs
	}
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return newConfigReadErr(file, err)
	}
	if err := viper.ReadConfig(bytes.NewReader(content)); err != nil {
		return newConfigReadErr(file, err)
	}
	return nil
}

func newConfigReadErr(file string, reason error) error {
	return fmt.Errorf("could not read the config from %v: %v", file, reason)
}

package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Contains all the keys for Wash's shared config
const (
	SocketKey = "socket"
)

// Socket is the path to the Wash server's UNIX
// socket
var Socket string

// Load Wash's config.
func Load() error {
	// Set any defaults
	viper.SetDefault(SocketKey, "/tmp/wash-api.sock")

	// Tell viper that the config. can be read from WASH_<entry>
	// environment variables
	viper.SetEnvPrefix("WASH")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// TODO: Add any additional config files, then make sure to
	// invoke viper.ReadInConfig() to read-in their values

	// Load the shared config
	Socket = viper.GetString(SocketKey)

	return nil
}

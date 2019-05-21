package server

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/fuse"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/plugin/aws"
	"github.com/puppetlabs/wash/plugin/docker"
	"github.com/puppetlabs/wash/plugin/kubernetes"

	log "github.com/sirupsen/logrus"
)

// Opts exposes additional configuration for server operation.
type Opts struct {
	CPUProfilePath      string
	ExternalPlugins     []plugin.ExternalPluginSpec
	LogFile             string
	// LogLevel can be "warn", "info", "debug", or "trace".
	LogLevel            string
}

type controlChannels struct {
	stopCh    chan<- context.Context
	stoppedCh <-chan struct{}
}

// Server encapsulates a running wash server with both Socket and FUSE servers.
type Server struct {
	mountpoint string
	socket     string
	opts       Opts
	logFH      *os.File
	api        controlChannels
	fuse       controlChannels
}

var levelMap = map[string]log.Level{
	"warn":  log.WarnLevel,
	"info":  log.InfoLevel,
	"debug": log.DebugLevel,
	"trace": log.TraceLevel,
}

// New creates a new Server.
func New(mountpoint string, socket string, opts Opts) *Server {
	return &Server{mountpoint: mountpoint, socket: socket, opts: opts}
}

// Start starts the server. It returns once the server is ready.
func (s *Server) Start() error {
	level, ok := levelMap[s.opts.LogLevel]
	if !ok {
		allLevels := make([]string, 0, len(levelMap))
		for level := range levelMap {
			allLevels = append(allLevels, level)
		}
		return fmt.Errorf("%v is not a valid level. Valid levels are %v", s.opts.LogLevel, strings.Join(allLevels, ", "))
	}

	log.SetLevel(level)
	if s.opts.LogFile != "" {
		logFH, err := os.Create(s.opts.LogFile)
		if err != nil {
			return err
		}

		log.SetOutput(logFH)
		s.logFH = logFH
	}

	registry := plugin.NewRegistry()
	loadInternalPlugins(registry)
	if len(s.opts.ExternalPlugins) > 0 {
		loadExternalPlugins(registry, s.opts.ExternalPlugins)
	}
	if len(registry.Plugins()) == 0 {
		return fmt.Errorf("No plugins loaded")
	}

	plugin.InitCache()

	apiServerStopCh, apiServerStoppedCh, err := api.StartAPI(registry, s.mountpoint, s.socket)
	if err != nil {
		return err
	}
	s.api = controlChannels{stopCh: apiServerStopCh, stoppedCh: apiServerStoppedCh}

	fuseServerStopCh, fuseServerStoppedCh, err := fuse.ServeFuseFS(registry, s.mountpoint)
	if err != nil {
		s.stopAPIServer()
		return err
	}
	s.fuse = controlChannels{stopCh: fuseServerStopCh, stoppedCh: fuseServerStoppedCh}

	if s.opts.CPUProfilePath != "" {
		f, err := os.Create(s.opts.CPUProfilePath)
		if err != nil {
			log.Fatal(err)
		}
		errz.Fatal(pprof.StartCPUProfile(f))
	}
	return nil
}

func (s *Server) stopAPIServer() {
	// Shutdown the API server; wait for the shutdown to finish
	apiShutdownDeadline := time.Now().Add(3 * time.Second)
	apiShutdownCtx, cancelFunc := context.WithDeadline(context.Background(), apiShutdownDeadline)
	defer cancelFunc()
	s.api.stopCh <- apiShutdownCtx
	close(s.api.stopCh)
	<-s.api.stoppedCh
}

func (s *Server) stopFUSEServer() {
	// Shutdown the FUSE server; wait for the shutdown to finish
	close(s.fuse.stopCh)
	<-s.fuse.stoppedCh
}

func (s *Server) shutdown() {
	if s.opts.CPUProfilePath != "" {
		pprof.StopCPUProfile()
	}

	// Close any open journals on shutdown to ensure remaining entries are flushed to disk.
	activity.CloseAll()

	if s.logFH != nil {
		s.logFH.Close()
	}
}

// Wait blocks until the server exits due to an error or a signal is delivered.
// Only one of Wait or Stop should be called.
func (s *Server) Wait(sigCh chan os.Signal) {
	select {
	case <-sigCh:
		s.stopAPIServer()
		s.stopFUSEServer()
	case <-s.fuse.stoppedCh:
		// This code-path is possible if the FUSE server prematurely shuts down, which
		// can happen if the user unmounts the mountpoint while the server's running.
		s.stopAPIServer()
	case <-s.api.stoppedCh:
		// This code-path is possible if the API server prematurely shuts down
		s.stopFUSEServer()
	}
	s.shutdown()
}

// Stop the server and any related activity. Only one of Wait or Stop should be called.
func (s *Server) Stop() {
	s.stopAPIServer()
	s.stopFUSEServer()
	s.shutdown()
}

func loadPlugin(registry *plugin.Registry, name string, root plugin.Root) {
	log.Infof("Loading %v", name)
	if err := registry.RegisterPlugin(root); err != nil {
		// %+v is a convention used by some errors to print additional context such as a stack trace
		log.Warnf("%v failed to load: %+v", name, err)
	}
}

func loadInternalPlugins(registry *plugin.Registry) {
	log.Debug("Loading internal plugins")
	loadPlugin(registry, "aws", &aws.Root{})
	loadPlugin(registry, "docker", &docker.Root{})
	loadPlugin(registry, "kubernetes", &kubernetes.Root{})
	log.Debug("Finished loading internal plugins")
}

func loadExternalPlugins(registry *plugin.Registry, externalPlugins []plugin.ExternalPluginSpec) {
	log.Infof("Loading external plugins")
	for _, p := range externalPlugins {
		log.Infof("Loading %v", p.Script)
		if err := registry.RegisterExternalPlugin(p); err != nil {
			// %+v is a convention used by some errors to print additional context such as a stack trace
			log.Warnf("%v failed to load: %+v", p.Script, err)
		}
	}
	log.Infof("Finished loading external plugins")
}

package server

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/analytics"
	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/fuse"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/plugin/aws"
	"github.com/puppetlabs/wash/plugin/docker"
	"github.com/puppetlabs/wash/plugin/gcp"
	"github.com/puppetlabs/wash/plugin/kubernetes"

	log "github.com/sirupsen/logrus"
)

var InternalPlugins = map[string]plugin.Root{
	"aws":        &aws.Root{},
	"docker":     &docker.Root{},
	"gcp":        &gcp.Root{},
	"kubernetes": &kubernetes.Root{},
}

// Opts exposes additional configuration for server operation.
type Opts struct {
	CPUProfilePath string
	LogFile        string
	// LogLevel can be "warn", "info", "debug", or "trace".
	LogLevel     string
	PluginConfig map[string]map[string]interface{}
}

// SetupLogging configures log level and output file according to configured options.
// If an output file was configured, returns a handle for you to close later.
func (o Opts) SetupLogging() (*os.File, error) {
	level, err := log.ParseLevel(o.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("%v is not a valid level; use warn, info, debug, trace", o.LogLevel)
	}

	log.SetLevel(level)
	if o.LogFile != "" {
		logFH, err := os.Create(o.LogFile)
		if err != nil {
			return nil, err
		}

		log.SetOutput(logFH)
		return logFH, nil
	}
	return nil, nil
}

type controlChannels struct {
	stopCh    chan<- context.Context
	stoppedCh <-chan struct{}
}

// Server encapsulates a running wash server with both Socket and FUSE servers.
type Server struct {
	mountpoint       string
	socket           string
	opts             Opts
	logFH            *os.File
	api              controlChannels
	fuse             controlChannels
	plugins          map[string]plugin.Root
	analyticsClient  analytics.Client
	forVerifyInstall bool
}

// New creates a new Server. Accepts a list of plugins to load.
func New(mountpoint string, socket string, plugins map[string]plugin.Root, opts Opts) *Server {
	return &Server{
		mountpoint: mountpoint,
		socket:     socket,
		plugins:    plugins,
		opts:       opts,
	}
}

// ForVerifyInstall creates a new Server that's meant to be used
// for verifying a given Wash installation.
func ForVerifyInstall(mountpoint string, socket string) *Server {
	return &Server{
		mountpoint:       mountpoint,
		socket:           socket,
		forVerifyInstall: true,
		opts: Opts{
			LogLevel: "warn",
		},
	}
}

// Start starts the server. It returns once the server is ready.
func (s *Server) Start() error {
	var err error
	if s.logFH, err = s.opts.SetupLogging(); err != nil {
		return err
	}

	registry := plugin.NewRegistry()

	if !s.forVerifyInstall {
		s.loadPlugins(registry)
		if len(registry.Plugins()) == 0 {
			return fmt.Errorf("No plugins loaded")
		}

		plugin.InitCache()

		analyticsConfig, err := analytics.GetConfig()
		if err != nil {
			return err
		}
		s.analyticsClient = analytics.NewClient(analyticsConfig)
	}

	apiServerStopCh, apiServerStoppedCh, err := api.StartAPI(
		registry,
		s.mountpoint,
		s.socket,
		s.analyticsClient,
	)
	if err != nil {
		return err
	}
	s.api = controlChannels{stopCh: apiServerStopCh, stoppedCh: apiServerStoppedCh}

	fuseServerStopCh, fuseServerStoppedCh, err := fuse.ServeFuseFS(
		registry,
		s.mountpoint,
		s.analyticsClient,
	)
	if err != nil {
		s.stopAPIServer()
		return err
	}
	s.fuse = controlChannels{stopCh: fuseServerStopCh, stoppedCh: fuseServerStoppedCh}

	if !s.forVerifyInstall {
		if s.opts.CPUProfilePath != "" {
			f, err := os.Create(s.opts.CPUProfilePath)
			if err != nil {
				log.Fatal(err)
			}
			errz.Fatal(pprof.StartCPUProfile(f))
		}

		// Submit the initial start-up ping to GA. It's OK to do this synchronously
		// because this is the first hit so the analytics client will not send it
		// over the network.
		if err := s.analyticsClient.Screenview("wash", analytics.Params{}); err != nil {
			log.Infof("Failed to submit the initial start-up ping: %v", err)
		}
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
	if s.forVerifyInstall {
		return
	}

	if s.opts.CPUProfilePath != "" {
		pprof.StopCPUProfile()
	}

	// Close any open journals on shutdown to ensure remaining entries are flushed to disk.
	activity.CloseAll()

	// Flush any outstanding analytics hits. We do this asynchronously
	// so that the server process isn't blocked on its cleanup (in case
	// the network is slow).
	doneCh := make(chan struct{})
	ticker := time.NewTicker(analytics.FlushDuration)
	defer ticker.Stop()
	go func() {
		s.analyticsClient.Flush()
		close(doneCh)
	}()
	select {
	case <-doneCh:
		// Pass-thru
	case <-ticker.C:
		// Pass-thru
	}

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

func (s *Server) loadPlugins(registry *plugin.Registry) {
	log.Debug("Loading plugins")
	var wg sync.WaitGroup
	var mux sync.Mutex
	var failedPlugins []string

	for name, root := range s.plugins {
		log.Infof("Loading %v", name)
		wg.Add(1)
		go func(name string, root plugin.Root) {
			if err := registry.RegisterPlugin(root, s.opts.PluginConfig[name]); err != nil {
				// %+v is a convention used by some errors to print additional context such as a stack trace
				log.Warnf("%v failed to load: %+v", name, err)
				if _, ok := InternalPlugins[name]; ok {
					mux.Lock()
					failedPlugins = append(failedPlugins, name)
					mux.Unlock()
				}
			}
			wg.Done()
		}(name, root)
	}

	wg.Wait()
	if len(failedPlugins) > 0 {
		log.Warnf(
			"You can use 'docs <plugin>' (e.g. 'docs %v') to view set-up instructions for %v. However, you'll need to exit then restart the shell in order to properly reload them.\n",
			failedPlugins[0],
			strings.Join(failedPlugins, ", "),
		)
	}
	log.Debug("Finished loading plugins")
}

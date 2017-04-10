package healthz

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type logger interface {
	Info(args ...interface{})
	Debug(args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

type HealthCheckable interface {
	HealthZ() error
}

type ProviderInfo struct {
	Check       HealthCheckable
	Description string
	Type        string
}

type Error struct {
	Type        string
	ErrMsg      string
	Description string
}

type HTTPResponse struct {
	Errors   []Error
	Hostname string
}

type Config struct {
	BindPort       int
	BindAddr       string
	Providers      []ProviderInfo
	Hostname       string
	Log            logger
	ServerErrorLog *log.Logger
}

type HealthChecker struct {
	providers []ProviderInfo
	Server    *http.Server
	hostname  string
	log       logger
}

func New(config Config) (*HealthChecker, error) {
	h := &HealthChecker{
		providers: config.Providers,
		log:       config.Log,
		hostname:  config.Hostname,
	}

	if h.log == nil {
		return nil, errors.New("required config option 'Log' not found")
	}

	// Hostname is sent in check results, so that we can tell which pod the health check ran on.
	if h.hostname == "" {
		var err error
		h.hostname, err = os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("could not detect hostname: %s", err.Error())
		}
		h.log.Info("autodetected hostname as: ", h.hostname)
	}

	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(h.HandleHealthz))
	mux.Handle("/liveness", http.HandlerFunc(h.HandleLiveness))
	h.Server = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", config.BindAddr, config.BindPort),
		ReadTimeout:    time.Second * 45,
		WriteTimeout:   time.Second * 45,
		MaxHeaderBytes: 1 << 20,
		Handler:        mux,
		ErrorLog:       config.ServerErrorLog,
	}
	return h, nil
}

// HandleHealthz is the http handler for `/healthz`
func (h *HealthChecker) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	resp := &HTTPResponse{
		Hostname: h.hostname,
	}

	// Check all our health providers
	for _, provider := range h.providers {
		err := provider.Check.HealthZ()
		if err != nil {
			resp.Errors = append(resp.Errors, Error{
				Type:        provider.Type,
				ErrMsg:      err.Error(),
				Description: provider.Description,
			})
		}
	}
	if len(resp.Errors) > 0 {
		for _, e := range resp.Errors {
			h.log.Errorf("Check failed: %s: %s, error: %s", e.Type, e.Description, e.ErrMsg)
		}
	} else {
		h.log.Debug("All checks passed")
	}
	enc := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")

	// We use result `200 OK` regardless of whether there were errors.
	// In Sensu, we check the body contains `"Errors":null`, and alert if not.
	// If we returned a 5xx status code, the body check would not be
	// executed and the check result would not contain the error text for on-call.
	w.WriteHeader(http.StatusOK)
	err := enc.Encode(resp)
	if err != nil {
		h.log.Error(err)
	}
}

// HandleLiveness is the http handler for `/liveness`
func (h *HealthChecker) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Liveness check: OK")
	w.Write([]byte("OK"))
}

// StartHealthz should be run in a new goroutine.
func (h *HealthChecker) StartHealthz() {
	h.log.Debug("Starting healthz server")
	err := h.Server.ListenAndServe()
	if err != nil {
		h.log.Error(err)
	}
}

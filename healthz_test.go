package healthz_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pantheon-systems/go-healthz"
)

var config healthz.Config

func init() {
	config = healthz.Config{
		BindPort: 80,
		BindAddr: "localhost",
		Hostname: "tester",
	}
}

func TestHappy(t *testing.T) {
	config.Providers = []healthz.ProviderInfo{
		healthz.ProviderInfo{
			Check: &Happy{},
		},
	}
	config.Log = testLogger{t: t}
	hz, err := healthz.New(config)
	fatalIfErr(t, err)

	req, err := http.NewRequest("GET", "/healthz", nil)
	fatalIfErr(t, err)
	w := httptest.NewRecorder()
	hz.Server.Handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("Expected 200 OK, got:", w.Code)
	}
	if w.Body.String() != `{"Errors":null,"Hostname":"tester"}`+"\n" {
		t.Fatal("Unexpected JSON body, got:", w.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	config.Providers = []healthz.ProviderInfo{
		healthz.ProviderInfo{
			Check: &Happy{},
		},
	}
	config.Log = testLogger{t: t}
	hz, err := healthz.New(config)
	fatalIfErr(t, err)

	req, err := http.NewRequest("GET", "/", nil)
	fatalIfErr(t, err)
	w := httptest.NewRecorder()
	hz.Server.Handler.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatal("Expected 404 Not Found, got:", w.Code)
	}
}

func TestUnhappy(t *testing.T) {
	config.Providers = []healthz.ProviderInfo{
		healthz.ProviderInfo{
			Check:       &Unhappy{},
			Type:        "DBConn",
			Description: "Ensure the database connection is up",
		},
	}
	config.Log = testLogger{t: t}
	hz, err := healthz.New(config)
	fatalIfErr(t, err)

	req, err := http.NewRequest("GET", "/healthz", nil)
	fatalIfErr(t, err)
	w := httptest.NewRecorder()
	hz.Server.Handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("Expected 200 OK, got:", w.Code)
	}
	if w.Body.String() != `{"Errors":[{"Type":"DBConn","ErrMsg":"failed","Description":"Ensure the database connection is up"}],"Hostname":"tester"}`+"\n" {
		t.Fatal("Unexpected JSON body, got:", w.Body.String())
	}
}

func TestMultipleOneUnhappy(t *testing.T) {
	config.Providers = []healthz.ProviderInfo{
		healthz.ProviderInfo{
			Check:       &Unhappy{},
			Type:        "DBConn",
			Description: "Ensure the database connection is up",
		},
		healthz.ProviderInfo{
			Check:       &Happy{},
			Type:        "Foo",
			Description: "Ensure we can reach Foo",
		},
		healthz.ProviderInfo{
			Check:       &Happy{},
			Type:        "Metric",
			Description: "Watch a key metric for failure",
		},
	}
	config.Log = testLogger{t: t}
	hz, err := healthz.New(config)
	fatalIfErr(t, err)

	req, err := http.NewRequest("GET", "/healthz", nil)
	fatalIfErr(t, err)
	w := httptest.NewRecorder()
	hz.Server.Handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("Expected 200 OK, got:", w.Code)
	}
	if w.Body.String() != `{"Errors":[{"Type":"DBConn","ErrMsg":"failed","Description":"Ensure the database connection is up"}],"Hostname":"tester"}`+"\n" {
		t.Fatal("Unexpected JSON body, got:", w.Body.String())
	}
}

func TestNoProviders(t *testing.T) {
	config.Providers = []healthz.ProviderInfo{}
	config.Log = testLogger{t: t}
	hz, err := healthz.New(config)
	fatalIfErr(t, err)

	req, err := http.NewRequest("GET", "/healthz", nil)
	fatalIfErr(t, err)
	w := httptest.NewRecorder()
	hz.Server.Handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("Expected 200 OK, got:", w.Code)
	}
	if w.Body.String() != `{"Errors":null,"Hostname":"tester"}`+"\n" {
		t.Fatal("Unexpected JSON body, got:", w.Body.String())
	}
}

type Happy struct{}

func (hz *Happy) HealthZ() error {
	return nil
}

type Unhappy struct{}

func (hz *Unhappy) HealthZ() error {
	return errors.New("failed")
}

func fatalIfErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

type testLogger struct {
	t *testing.T
}

func (tl testLogger) Info(args ...interface{}) {
	tl.t.Log("INFO:", args)
}
func (tl testLogger) Debug(args ...interface{}) {
	tl.t.Log("DEBUG:", args)
}
func (tl testLogger) Error(args ...interface{}) {
	tl.t.Log("ERROR:", args)
}
func (tl testLogger) Errorf(format string, args ...interface{}) {
	tl.t.Logf("ERRORF: "+format, args...)
}

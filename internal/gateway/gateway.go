package gateway

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/ITResourcesOSS/sgulgate/internal/config"
)

type apiDefinition struct {
	name      string
	path      string
	balancing string
	endpoints []string
}

// Gateway .
type Gateway struct {
	api map[string]apiDefinition
}

// New returns a new instance of the Gateway struct.
func New() Gateway {
	gw := Gateway{api: make(map[string]apiDefinition)}
	apiConf := config.Config.API
	log.Printf("configuring %s definitions", apiConf.Name)
	for _, endpoint := range apiConf.Endpoints {
		path := fmt.Sprintf("%s/v%s", endpoint.Path, endpoint.Version)
		apiDef := apiDefinition{
			name:      endpoint.Name,
			path:      path,
			balancing: endpoint.Proxy.Balancing.Strategy,
			endpoints: make([]string, 0),
		}

		for _, target := range endpoint.Proxy.Targets {
			apiDef.endpoints = append(
				apiDef.endpoints,
				fmt.Sprintf("%s://%s%s", target.Schema, target.Host, target.Path))
		}
		gw.api[path] = apiDef

		log.Printf("endpoint name: %s - path: %s - targets: %+v", apiDef.name, apiDef.path, apiDef.endpoints)
	}

	return gw
}

// PrintConfiguration .
func (gw Gateway) PrintConfiguration() {
	log.Printf("Gateway Configuation: %+v\n", config.Config)
}

// Start starts the Gateway starting the http server on configured endppoint.
func (gw Gateway) Start() {
	log.Println("starting Gateway...")
	log.Printf("gateway endpoint: %s", gw.endpointPath())

	http.HandleFunc(gw.endpointPath(), func(w http.ResponseWriter, req *http.Request) {
		req.URL.Path = gw.stripPath(req.URL.Path)
		name, version, err := gw.GetNameAndVersion(req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("Name: %s", name)
		log.Printf("Version: %s", version)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	gw.serve()
}

func (gw Gateway) serve() {
	log.Printf("endpoint started and listening on localhost:9000%s", config.Config.Gateway.Endpoint.Path)
	log.Fatal(http.ListenAndServe(":9000", nil))
}

// GetNameAndVersion .
func (gw Gateway) GetNameAndVersion(target *url.URL) (name, version string, err error) {
	path := target.Path
	if len(path) > 1 && path[0] == '/' {
		path = path[1:]
	}
	tmp := strings.Split(path, "/")
	if len(tmp) < 2 {
		return "", "", fmt.Errorf("Invalid path")
	}
	name, version = tmp[0], tmp[1]
	target.Path = "/" + strings.Join(tmp[2:], "/")
	return name, version, nil
}

func sanitizePath(path string) string {
	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}
	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func (gw Gateway) endpointPath() string {
	epath := sanitizePath(config.Config.Gateway.Endpoint.Path)
	return fmt.Sprintf("/%s/", epath)
}

func (gw Gateway) stripPath(path string) string {
	epath := sanitizePath(config.Config.Gateway.Endpoint.Path)
	return strings.Replace(path, fmt.Sprintf("/%s", epath), "", -1)
}
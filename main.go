package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"void/extras"
	"void/state"
	"void/types"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/infinitybotlist/eureka/zapchi"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed services
	servicesFS embed.FS
	//go:embed app.html
	appHTML []byte
	//go:embed error.html
	errorHTML []byte
	//go:embed assets/*
	assets embed.FS

	appTemplate   *template.Template
	errorTemplate *template.Template
)

// Load the services directory into the state
func init() {
	state.Init()

	// Initialize service registry
	state.Registry = types.ServiceRegistry{
		Services: make([]types.Service, 0),
		APIUrls:  make([]string, 0),
	}

	// First load default.yaml
	defaultConfigBytes, err := servicesFS.ReadFile("services/default.yaml")
	if err != nil {
		log.Fatal("Could not read default.yaml:", err)
	}

	err = yaml.Unmarshal(defaultConfigBytes, &state.Registry.DefaultService)
	if err != nil {
		log.Fatal("Could not parse default.yaml:", err)
	}

	// Validate default service
	validator := validator.New()
	err = validator.Struct(state.Registry.DefaultService)
	if err != nil {
		log.Fatal("Could not validate default.yaml:", err)
	}

	// Load all other service configs
	entries, err := servicesFS.ReadDir("services")
	if err != nil {
		log.Fatal("Could not read services directory:", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "default.yaml" {
			continue
		}

		filePath := filepath.Join("services", entry.Name())
		fileBytes, err := servicesFS.ReadFile(filePath)
		if err != nil {
			state.Logger.Error("Could not read service file:", filePath, err)
			continue
		}

		var config types.ServiceConfig
		err = yaml.Unmarshal(fileBytes, &config)
		if err != nil {
			state.Logger.Error("Could not parse service file:", filePath, err)
			continue
		}

		// Validate config
		err = validator.Struct(config)
		if err != nil {
			state.Logger.Error("Could not validate service file:", filePath, err)
			continue
		}

		// Add services and API URLs to registry
		state.Registry.Services = append(state.Registry.Services, config.Services...)
		state.Registry.APIUrls = append(state.Registry.APIUrls, config.APIUrls...)

		state.Logger.Info("Loaded service file:", filePath)
	}

	appTemplate = template.Must(template.New("app").Parse(string(appHTML)))
	errorTemplate = template.Must(template.New("error").Parse(string(errorHTML)))

	state.Logger.Info("Loaded services:", len(state.Registry.Services))
	state.Logger.Info("Loaded API URLs:", len(state.Registry.APIUrls))

	// Set version if built with -ldflags
	extras.SetVersion(extras.GetVoidInfo().Version)
}

func dataHandlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// limit body to 10mb
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

		var allowedHeaders = []string{"Content-Type", "Authorization"}

		reqHeaderList := strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",")

		for _, name := range reqHeaderList {
			if name == "" {
				continue
			}

			state.Logger.Info(name)

			if strings.HasPrefix(strings.ToLower(name), "x-") {
				allowedHeaders = append(allowedHeaders, strings.ReplaceAll(name, " ", ""))
			}
		}

		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if r.URL.Query().Get("data") != "" {
			split := strings.Split(r.URL.Query().Get("data"), "|")

			if len(split) > 0 {
				r.Method = split[0]
			}

			if len(split) > 1 {
				// All the remaining sections are considered PATH
				r.URL.Path = strings.Join(split[1:], "/")
			}
		}

		// Handle options immediately here
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// serveErrorPage renders the appropriate error page
func serveErrorPage(w http.ResponseWriter, r *http.Request, statusCode int, errorTitle, errorMsg, errorDetails string, service types.Service) {
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "text/html")

	errorCtx := struct {
		StatusCode     int
		ErrorTitle     string
		ErrorMessage   string
		ErrorDetails   string
		MatchedService types.Service
		Path           string
		Hostname       string
		Info           extras.VoidInfo
		Redirect       string
		ErrorSummary   string
		ErrorStack     string
		ErrorTime      string
	}{
		StatusCode:     statusCode,
		ErrorTitle:     errorTitle,
		ErrorMessage:   errorMsg,
		ErrorDetails:   errorDetails,
		MatchedService: service,
		Path:           r.URL.Path,
		Hostname:       r.Host,
		Info:           extras.GetVoidInfo(),
		Redirect:       r.URL.Query().Get("src"),
		ErrorSummary:   "Error connecting to backend: " + service.Host,
		ErrorStack:     captureErrorStack(fmt.Errorf("%s", errorMsg)),
	}

	err := errorTemplate.Execute(w, errorCtx)
	if err != nil {
		state.Logger.Error("Could not execute error template:", err)
		w.Write([]byte(err.Error()))
	}
}

func setupRoutes() {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(
		middleware.Recoverer,
		dataHandlerMiddleware,
		middleware.RealIP,
		middleware.CleanPath,
		zapchi.Logger(state.Logger, "api"),
		middleware.Timeout(30*time.Second),
	)

	subbedAssets, err := fs.Sub(assets, "assets")

	if err != nil {
		panic(err)
	}

	r.HandleFunc("/__voidStatic/*", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/__voidStatic", http.FileServer(http.FS(subbedAssets))).ServeHTTP(w, r)
	})

	// Health check endpoints (use handler from health.go)
	r.HandleFunc("/health", extras.HealthHandler)
	r.HandleFunc("/healthz", extras.HealthHandler)

	// Update check endpoint
	r.HandleFunc("/update-check", extras.UpdateCheckHandler)

	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		// Get the hostname from the request
		hostname := r.Host

		// Also get the root domain from the request
		var rootDomain string

		rootDomainLst := strings.Split(hostname, ".")

		// Then slice last 2 elements IF the length is greater than 2
		if len(rootDomainLst) > 2 {
			rootDomain = strings.Join(rootDomainLst[len(rootDomainLst)-2:], ".")
		} else {
			rootDomain = hostname
		}

		// Find the right service from config
		var service = types.Service{
			Name:    state.Registry.DefaultService.Name,
			Domain:  state.Registry.DefaultService.Domain,
			Support: state.Registry.DefaultService.Support,
			Status:  state.Registry.DefaultService.Status,
		}
		var matched bool
		var backendURL string

		for _, s := range state.Registry.Services {
			if s.Domain == rootDomain {
				service = s
				matched = true
				backendURL = s.Host // Use the host field as backend URL
				break
			}
		}

		// If this is an API URL, serve the maintenance JSON as before
		if slices.Contains(state.Registry.APIUrls, hostname) {
			w.WriteHeader(http.StatusRequestTimeout)
			w.Header().Set("Content-Type", "application/json")

			apiCtx := types.APICtx{
				Message: "This service is down for maintenance...",
				Service: service,
				Info:    extras.GetVoidInfo(),
			}

			json.NewEncoder(w).Encode(apiCtx)
			return
		}

		// If a matching service is found, reverse proxy to its backend
		if matched && backendURL != "" {
			target, err := url.Parse(backendURL)
			if err != nil {
				state.Logger.Error("Invalid backend URL for service:", service.Name, backendURL, err)
				serveErrorPage(w, r, http.StatusBadGateway, "Bad Gateway",
					"The backend URL for this service is invalid or misconfigured, this error 100% means that the owner of this service misconfigured void.",
					fmt.Sprintf("Error parsing backend URL: %s\nDetails: %v", backendURL, err), service)
				return
			}

			proxy := httputil.NewSingleHostReverseProxy(target)
			// Optionally, log proxy errors and return a meaningful error to the client
			originalDirector := proxy.Director
			proxy.Director = func(req *http.Request) {
				originalDirector(req)
				// Preserve original host header for backend
				req.Host = target.Host
			}
			proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
				state.Logger.Error("Proxy error for service:", service.Name, err)
				serveErrorPage(rw, req, http.StatusServiceUnavailable, "Service Unavailable",
					"The backend service is currently unreachable or experiencing issues, this error likely indicates that void was unable to proxy the service and ny errors in the debug section should confirm this.",
					fmt.Sprintf("Error connecting to backend: %s\nDetails: %v", backendURL, err), service)
			}
			proxy.ServeHTTP(w, r)
			return
		}

		// Fallback: serve the info/status page as before
		htmlCtx := types.HTMLCtx{
			MatchedService: service,
			Path:           r.URL.Path,
			Hostname:       hostname,
			Info:           extras.GetVoidInfo(),
			Redirect:       r.URL.Query().Get("src"),
		}

		// Set status code of 408
		w.WriteHeader(http.StatusRequestTimeout)
		w.Header().Add("Content-Type", "text/html")

		// Execute the template
		err := appTemplate.Execute(w, htmlCtx)

		if err != nil {
			state.Logger.Error("Could not execute template:", err)
			w.Write([]byte(err.Error()))
		}
	})

	err = http.ListenAndServe(":1292", r)

	if err != nil {
		panic(err)
	}
}

func main() {
	setupRoutes()
}

// Helper function to capture stack trace if available
func captureErrorStack(err error) string {
	// Check for errors that contain stack traces (like from github.com/pkg/errors)
	type stackTracer interface {
		StackTrace() []uintptr
	}

	// Try different error wrapping styles
	if st, ok := err.(stackTracer); ok {
		return fmt.Sprintf("%+v", st.StackTrace())
	}

	// For errors that might have their stack as a method or field
	v := reflect.ValueOf(err)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	// Try to get Stack method
	stackMethod := v.MethodByName("Stack")
	if stackMethod.IsValid() {
		stack := stackMethod.Call(nil)
		if len(stack) > 0 {
			return fmt.Sprintf("%v", stack[0].Interface())
		}
	}

	// Look for Stack field
	stackField := v.FieldByName("Stack")
	if stackField.IsValid() {
		return fmt.Sprintf("%v", stackField.Interface())
	}

	// Fallback to error message
	return ""
}

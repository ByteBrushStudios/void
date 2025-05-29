// Package types provides a service struct
package types

type Service struct {
	// Name is the name of the service
	Name string `yaml:"name" validate:"required"`
	// Host is the backend URL of the service
	Host string `yaml:"host" validate:"required,url"`
	// Domain is the domain of the service
	Domain string `yaml:"domain" validate:"required,hostname"`
	// Support is the support server/location of the service
	Support string `yaml:"support" validate:"required"`
	// Status is the status page of the service
	Status string `yaml:"status" validate:"required,url"`
}

// DefaultService represents the default service configuration
type DefaultService struct {
	Name    string `yaml:"name" validate:"required"`
	Domain  string `yaml:"domain" validate:"required,hostname"`
	Support string `yaml:"support" validate:"required"`
	Status  string `yaml:"status" validate:"required,url"`
}

// ServiceConfig represents a single service configuration file
type ServiceConfig struct {
	Services []Service `yaml:"services"`
	APIUrls  []string  `yaml:"apiUrls"`
}

// ServiceRegistry holds all loaded services and API URLs
type ServiceRegistry struct {
	Services       []Service
	APIUrls        []string
	DefaultService DefaultService
}

// The yaml document
type Document struct {
	Services []Service `yaml:"services"`
	APIUrls  []string  `yaml:"apiUrls"`
}

type HTMLCtx struct {
	MatchedService Service
	Path           string
	Hostname       string
	Info           interface{} // Use interface{} to allow extras.VoidInfo
	Redirect       string
}

type APICtx struct {
	Message string      `json:"message"`
	Service Service     `json:"service"`
	Info    interface{} `json:"info"`
}

type ErrorCtx struct {
	StatusCode     int
	ErrorTitle     string
	ErrorMessage   string
	ErrorDetails   string
	MatchedService Service
	Path           string
	Hostname       string
	Info           interface{}
	Redirect       string
}

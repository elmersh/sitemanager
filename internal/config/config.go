package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/elmersh/sitemanager/internal/utils"
	"gopkg.in/yaml.v3"
)

// Config representa la configuración del sitemanager
type Config struct {
	NginxPath          string            `yaml:"nginxPath"`
	SitesAvailable     string            `yaml:"sitesAvailable"`
	SitesEnabled       string            `yaml:"sitesEnabled"`
	DefaultUser        string            `yaml:"defaultUser"`
	DefaultGroup       string            `yaml:"defaultGroup"`
	PHPVersions        []string          `yaml:"phpVersions"`
	DefaultTemplate    string            `yaml:"defaultTemplate"`
	Templates          map[string]string `yaml:"templates"`
	SubdomainTemplates map[string]string `yaml:"subdomainTemplates"`
	SkelDir            string            `yaml:"skelDir"`
}

// LoadConfig carga la configuración desde el archivo de configuración
func LoadConfig() (*Config, error) {
	// Obtener el usuario actual para encontrar el directorio home
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener el usuario actual: %v", err)
	}

	// Buscar en la ruta de configuración
	configPath := filepath.Join(currentUser.HomeDir, ".config", "sitemanager.yaml")

	// Verificar si el archivo existe
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Si no existe, crear un archivo de configuración por defecto
		return createDefaultConfig(configPath)
	}

	// Leer el archivo de configuración
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el archivo de configuración: %v", err)
	}

	// Decodificar la configuración
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("no se pudo decodificar la configuración: %v", err)
	}

	// Inicializar el directorio skel si no existe
	if err := utils.InitSkelDir(cfg.SkelDir); err != nil {
		return nil, fmt.Errorf("error al inicializar el directorio skel: %v", err)
	}

	return &cfg, nil
}

// createDefaultConfig crea un archivo de configuración por defecto
func createDefaultConfig(path string) (*Config, error) {
	cfg := Config{
		NginxPath:       "/etc/nginx",
		SitesAvailable:  "/etc/nginx/sites-available",
		SitesEnabled:    "/etc/nginx/sites-enabled",
		DefaultUser:     "www-data",
		DefaultGroup:    "www-data",
		PHPVersions:     []string{"7.4", "8.0", "8.1", "8.2", "8.3", "8.4"},
		DefaultTemplate: "laravel",
		Templates: map[string]string{
			"laravel": "nginx/laravel.conf.tmpl",
			"nodejs":  "nginx/nodejs.conf.tmpl",
			"static":  "nginx/static.conf.tmpl",
		},
		SubdomainTemplates: map[string]string{
			"laravel": "nginx/subdomain_laravel.conf.tmpl",
			"nodejs":  "nginx/subdomain_nodejs.conf.tmpl",
			"static":  "nginx/subdomain_static.conf.tmpl",
		},
		SkelDir: "/etc/sitemanager/skel",
	}

	// Crear el directorio .config si no existe
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("no se pudo crear el directorio de configuración: %v", err)
	}

	// Convertir la configuración a YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("no se pudo codificar la configuración: %v", err)
	}

	// Escribir el archivo de configuración
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("no se pudo escribir el archivo de configuración: %v", err)
	}

	fmt.Printf("Se ha creado un archivo de configuración por defecto en %s\n", path)
	return &cfg, nil
}

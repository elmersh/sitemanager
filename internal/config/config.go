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
	// Configuración de sistema
	NginxPath          string            `yaml:"nginx_path"`
	SitesAvailable     string            `yaml:"sites_available"`
	SitesEnabled       string            `yaml:"sites_enabled"`
	DefaultUser        string            `yaml:"default_user"`
	DefaultGroup       string            `yaml:"default_group"`
	SkelDir            string            `yaml:"skel_dir"`
	
	// Configuración de usuario
	Email              string            `yaml:"email"`
	DefaultPHP         string            `yaml:"default_php"`
	DefaultPort        int               `yaml:"default_port"`
	
	// SSL/Certificados
	AgreeTOS           bool              `yaml:"agree_tos"`
	ForceRenewal       bool              `yaml:"force_renewal"`
	UseStaging         bool              `yaml:"use_staging"`
	UseWWW             bool              `yaml:"use_www"`
	
	// Backup y mantenimiento
	BackupConfigs      bool              `yaml:"backup_configs"`
	AutoUpdate         bool              `yaml:"auto_update"`
	CheckUpdates       bool              `yaml:"check_updates"`
	
	// Versiones y templates
	PHPVersions        []string          `yaml:"php_versions"`
	NodeVersions       []string          `yaml:"node_versions"`
	DefaultTemplate    string            `yaml:"default_template"`
	Templates          map[string]string `yaml:"templates"`
	SubdomainTemplates map[string]string `yaml:"subdomain_templates"`
	
	// Configuraciones avanzadas
	MaxSites           int               `yaml:"max_sites"`
	PortRange          PortRange         `yaml:"port_range"`
	DatabaseEngines    []string          `yaml:"database_engines"`
}

// PortRange define el rango de puertos disponibles para aplicaciones Node.js
type PortRange struct {
	Start int `yaml:"start"`
	End   int `yaml:"end"`
}

// LoadConfig carga la configuración desde el archivo de configuración
func LoadConfig() (*Config, error) {
	// Obtener el usuario actual para encontrar el directorio home
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener el usuario actual: %v", err)
	}

	// Buscar en la ruta de configuración
	configPath := filepath.Join(currentUser.HomeDir, ".config", "sitemanager", "config.yaml")

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

	// Aplicar valores por defecto para campos críticos faltantes
	applyDefaults(&cfg)

	// Inicializar el directorio skel si no existe
	if err := utils.InitSkelDir(cfg.SkelDir); err != nil {
		return nil, fmt.Errorf("error al inicializar el directorio skel: %v", err)
	}

	return &cfg, nil
}

// createDefaultConfig crea un archivo de configuración por defecto
func createDefaultConfig(path string) (*Config, error) {
	cfg := Config{
		// Configuración de sistema
		NginxPath:       "/etc/nginx",
		SitesAvailable:  "/etc/nginx/sites-available",
		SitesEnabled:    "/etc/nginx/sites-enabled",
		DefaultUser:     "www-data",
		DefaultGroup:    "www-data",
		SkelDir:         "/etc/sitemanager/skel",
		
		// Configuración de usuario (valores vacíos para que el usuario los configure)
		Email:           "", // Se debe configurar antes de usar SSL
		DefaultPHP:      "8.3", // PHP 8.3 por defecto como solicitaste
		DefaultPort:     3000,
		
		// SSL/Certificados
		AgreeTOS:        false, // Debe ser configurado por el usuario
		ForceRenewal:    false,
		UseStaging:      false, // Usar producción por defecto
		UseWWW:          true,
		
		// Backup y mantenimiento
		BackupConfigs:   true,
		AutoUpdate:      false, // Disabled por defecto para seguridad
		CheckUpdates:    true,  // Verificar actualizaciones por defecto
		
		// Versiones soportadas
		PHPVersions:     []string{"8.0", "8.1", "8.2", "8.3", "8.4"},
		NodeVersions:    []string{"16", "18", "20", "22"},
		DefaultTemplate: "laravel",
		
		// Templates
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
		
		// Configuraciones avanzadas
		MaxSites:        100, // Límite de sitios por servidor
		PortRange: PortRange{
			Start: 3000,
			End:   3999,
		},
		DatabaseEngines: []string{"postgresql", "mysql", "mongodb"},
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
	fmt.Printf("\n⚠️  IMPORTANTE: Configura tu email antes de usar SSL:\n")
	fmt.Printf("   Edita: %s\n", path)
	fmt.Printf("   Establece: email: tu@email.com\n")
	fmt.Printf("   Establece: agree_tos: true\n\n")
	
	return &cfg, nil
}

// SaveConfig guarda la configuración actual en el archivo
func (c *Config) SaveConfig() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("no se pudo obtener el usuario actual: %v", err)
	}
	
	configPath := filepath.Join(currentUser.HomeDir, ".config", "sitemanager", "config.yaml")
	
	// Convertir la configuración a YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("no se pudo codificar la configuración: %v", err)
	}
	
	// Escribir el archivo de configuración
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("no se pudo escribir el archivo de configuración: %v", err)
	}
	
	return nil
}

// ValidateSSLConfig verifica que la configuración SSL esté completa
func (c *Config) ValidateSSLConfig() error {
	if c.Email == "" {
		return fmt.Errorf("email no configurado: edita ~/.config/sitemanager/config.yaml y establece tu email")
	}
	
	if !c.AgreeTOS {
		return fmt.Errorf("términos de servicio no aceptados: establece agree_tos: true en la configuración")
	}
	
	return nil
}

// GetNextPort obtiene el siguiente puerto disponible en el rango configurado
func (c *Config) GetNextPort() int {
	// Implementación simple - en producción se debería verificar qué puertos están en uso
	return c.DefaultPort
}

// IsValidPHPVersion verifica si una versión de PHP está soportada
func (c *Config) IsValidPHPVersion(version string) bool {
	for _, v := range c.PHPVersions {
		if v == version {
			return true
		}
	}
	return false
}

// GetConfigPath obtiene la ruta del archivo de configuración
func GetConfigPath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("no se pudo obtener el usuario actual: %v", err)
	}
	
	return filepath.Join(currentUser.HomeDir, ".config", "sitemanager", "config.yaml"), nil
}

// applyDefaults aplica valores por defecto a campos críticos que puedan estar vacíos
func applyDefaults(cfg *Config) {
	// Configuración de sistema
	if cfg.NginxPath == "" {
		cfg.NginxPath = "/etc/nginx"
	}
	if cfg.SitesAvailable == "" {
		cfg.SitesAvailable = "/etc/nginx/sites-available"
	}
	if cfg.SitesEnabled == "" {
		cfg.SitesEnabled = "/etc/nginx/sites-enabled"
	}
	if cfg.DefaultUser == "" {
		cfg.DefaultUser = "www-data"
	}
	if cfg.DefaultGroup == "" {
		cfg.DefaultGroup = "www-data"
	}
	if cfg.SkelDir == "" {
		cfg.SkelDir = "/etc/sitemanager/skel"
	}
	
	// Configuración de usuario
	if cfg.DefaultPHP == "" {
		cfg.DefaultPHP = "8.3"
	}
	if cfg.DefaultPort == 0 {
		cfg.DefaultPort = 3000
	}
	
	// Configuración por defecto de templates si está vacío
	if cfg.DefaultTemplate == "" {
		cfg.DefaultTemplate = "laravel"
	}
	
	// Inicializar maps si están nil
	if cfg.Templates == nil {
		cfg.Templates = map[string]string{
			"laravel": "nginx/laravel.conf.tmpl",
			"nodejs":  "nginx/nodejs.conf.tmpl",
			"static":  "nginx/static.conf.tmpl",
		}
	}
	
	if cfg.SubdomainTemplates == nil {
		cfg.SubdomainTemplates = map[string]string{
			"laravel": "nginx/subdomain_laravel.conf.tmpl",
			"nodejs":  "nginx/subdomain_nodejs.conf.tmpl",
			"static":  "nginx/subdomain_static.conf.tmpl",
		}
	}
	
	// Versiones soportadas
	if cfg.PHPVersions == nil || len(cfg.PHPVersions) == 0 {
		cfg.PHPVersions = []string{"8.0", "8.1", "8.2", "8.3", "8.4"}
	}
	if cfg.NodeVersions == nil || len(cfg.NodeVersions) == 0 {
		cfg.NodeVersions = []string{"16", "18", "20", "22"}
	}
	
	// Configuraciones avanzadas
	if cfg.MaxSites == 0 {
		cfg.MaxSites = 100
	}
	if cfg.PortRange.Start == 0 {
		cfg.PortRange.Start = 3000
	}
	if cfg.PortRange.End == 0 {
		cfg.PortRange.End = 3999
	}
	if cfg.DatabaseEngines == nil || len(cfg.DatabaseEngines) == 0 {
		cfg.DatabaseEngines = []string{"postgresql", "mysql", "mongodb"}
	}
}

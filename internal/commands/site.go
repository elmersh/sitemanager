package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/elmersh/sitemanager/internal/config"
	"github.com/elmersh/sitemanager/internal/utils"
	"github.com/spf13/cobra"
)

// SiteOptions contiene las opciones para el comando site
type SiteOptions struct {
	Domain   string
	Type     string
	PHP      string
	Port     int
	User     string
	HomeDir  string
	NginxDir string
}

// AddSiteCommand agrega el comando site al comando raíz
func AddSiteCommand(rootCmd *cobra.Command, cfg *config.Config) {
	// Opciones del comando
	var opts SiteOptions
	var port int

	// Crear comando site
	siteCmd := &cobra.Command{
		Use:   "site",
		Short: "Configurar un nuevo sitio web",
		Long:  `Configura un nuevo sitio web creando un usuario, directorios y configuración de Nginx.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Configurar opciones
			if opts.Domain == "" {
				return fmt.Errorf("el dominio es obligatorio")
			}

			// Usar valores por defecto si no se especifican
			if opts.Type == "" {
				opts.Type = cfg.DefaultTemplate
			}

			// Verificar que el tipo de sitio es válido
			if _, ok := cfg.Templates[opts.Type]; !ok {
				return fmt.Errorf("tipo de sitio no válido: %s", opts.Type)
			}

			// Configurar usuario y directorios
			opts.User = strings.Split(opts.Domain, ".")[0]
			opts.HomeDir = filepath.Join("/home", opts.Domain)
			opts.NginxDir = filepath.Join(opts.HomeDir, ".nginx")
			opts.Port = port

			// Crear usuario y directorios
			if err := createUserAndDirs(&opts); err != nil {
				return err
			}

			// Generar configuración de Nginx
			if err := generateNginxConfig(&opts, cfg); err != nil {
				return err
			}

			// Crear enlaces simbólicos
			if err := createSymlinks(&opts, cfg); err != nil {
				return err
			}

			// Recargar configuración de Nginx
			if err := reloadNginx(); err != nil {
				return err
			}

			fmt.Printf("Sitio %s configurado correctamente\n", opts.Domain)
			return nil
		},
	}

	// Agregar flags
	siteCmd.Flags().StringVarP(&opts.Domain, "domain", "d", "", "Dominio del sitio (obligatorio)")
	siteCmd.Flags().StringVarP(&opts.Type, "type", "t", "", "Tipo de sitio (laravel, nodejs)")
	siteCmd.Flags().StringVarP(&opts.PHP, "php", "p", "8.1", "Versión de PHP (para sitios Laravel)")
	siteCmd.Flags().IntVarP(&port, "port", "P", 3000, "Puerto (para sitios Node.js)")

	// Marcar flags obligatorios
	siteCmd.MarkFlagRequired("domain")

	// Agregar comando al comando raíz
	rootCmd.AddCommand(siteCmd)
}

// createUserAndDirs crea el usuario y los directorios necesarios
func createUserAndDirs(opts *SiteOptions) error {
	// Verificar si el usuario ya existe
	if _, err := exec.Command("id", opts.User).Output(); err == nil {
		fmt.Printf("Usuario %s ya existe\n", opts.User)
	} else {
		// Crear usuario
		fmt.Printf("Creando usuario %s...\n", opts.User)
		cmd := exec.Command("useradd", "-m", "-d", opts.HomeDir, "-s", "/bin/bash", opts.User)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error al crear usuario: %v\n%s", err, output)
		}
	}

	// Crear directorio .nginx si no existe
	if err := os.MkdirAll(opts.NginxDir, 0755); err != nil {
		return fmt.Errorf("error al crear directorio Nginx: %v", err)
	}

	// Crear estructura básica del sitio
	publicDir := filepath.Join(opts.HomeDir, "public_html")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		return fmt.Errorf("error al crear directorio public_html: %v", err)
	}

	// Crear index.html de prueba
	indexFile := filepath.Join(publicDir, "index.html")
	indexContent := fmt.Sprintf("<html><body><h1>Bienvenido a %s</h1><p>Sitio configurado con SiteManager</p></body></html>", opts.Domain)
	if err := os.WriteFile(indexFile, []byte(indexContent), 0644); err != nil {
		return fmt.Errorf("error al crear archivo index.html: %v", err)
	}

	// Cambiar propietario de los directorios
	cmd := exec.Command("chown", "-R", fmt.Sprintf("%s:%s", opts.User, opts.User), opts.HomeDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al cambiar propietario: %v\n%s", err, output)
	}

	return nil
}

// generateNginxConfig genera la configuración de Nginx para el sitio
func generateNginxConfig(opts *SiteOptions, cfg *config.Config) error {
	// Leer plantilla según el tipo de sitio
	tmplPath := cfg.Templates[opts.Type]
	tmplContent, err := utils.ReadTemplateFile(tmplPath)
	if err != nil {
		return err
	}

	// Crear plantilla
	tmpl, err := template.New("nginx").Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("error al parsear plantilla: %v", err)
	}

	// Datos para la plantilla
	data := map[string]interface{}{
		"Domain":   opts.Domain,
		"RootDir":  filepath.Join(opts.HomeDir, "public_html"),
		"PHP":      opts.PHP,
		"Port":     opts.Port,
		"User":     opts.User,
		"HomeDir":  opts.HomeDir,
		"NginxDir": opts.NginxDir,
	}

	// Archivo de configuración
	confFile := filepath.Join(opts.NginxDir, fmt.Sprintf("%s.conf", opts.Domain))
	file, err := os.Create(confFile)
	if err != nil {
		return fmt.Errorf("error al crear archivo de configuración: %v", err)
	}
	defer file.Close()

	// Ejecutar plantilla
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("error al ejecutar plantilla: %v", err)
	}

	fmt.Printf("Configuración de Nginx generada en %s\n", confFile)
	return nil
}

// createSymlinks crea los enlaces simbólicos en los directorios de Nginx
func createSymlinks(opts *SiteOptions, cfg *config.Config) error {
	// Origen
	confFile := filepath.Join(opts.NginxDir, fmt.Sprintf("%s.conf", opts.Domain))

	// Destino en sites-available
	availableLink := filepath.Join(cfg.SitesAvailable, fmt.Sprintf("%s.conf", opts.Domain))

	// Eliminar enlace existente si existe
	os.Remove(availableLink)

	// Crear enlace en sites-available
	if err := os.Symlink(confFile, availableLink); err != nil {
		return fmt.Errorf("error al crear enlace en sites-available: %v", err)
	}

	// Destino en sites-enabled
	enabledLink := filepath.Join(cfg.SitesEnabled, fmt.Sprintf("%s.conf", opts.Domain))

	// Eliminar enlace existente si existe
	os.Remove(enabledLink)

	// Crear enlace en sites-enabled
	if err := os.Symlink(confFile, enabledLink); err != nil {
		return fmt.Errorf("error al crear enlace en sites-enabled: %v", err)
	}

	fmt.Println("Enlaces simbólicos creados correctamente")
	return nil
}

// reloadNginx recarga la configuración de Nginx
func reloadNginx() error {
	fmt.Println("Recargando configuración de Nginx...")
	cmd := exec.Command("systemctl", "reload", "nginx")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al recargar Nginx: %v\n%s", err, output)
	}
	return nil
}

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

// SecureOptions contiene las opciones para el comando secure
type SecureOptions struct {
	Domain  string
	Email   string
	User    string
	HomeDir string
	Force   bool
}

// AddSecureCommand agrega el comando secure al comando raíz
func AddSecureCommand(rootCmd *cobra.Command, cfg *config.Config) {
	// Opciones del comando
	var opts SecureOptions

	// Crear comando secure
	secureCmd := &cobra.Command{
		Use:   "secure",
		Short: "Configurar SSL para un sitio web",
		Long:  `Configura SSL para un sitio web usando Certbot y actualiza la configuración de Nginx.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Cargar configuración si no se ha pasado
			if cfg == nil {
				var err error
				cfg, err = config.LoadConfig()
				if err != nil {
					return fmt.Errorf("error al cargar la configuración: %v", err)
				}
			}

			// Verificar requisitos básicos del sistema
			if err := utils.CheckBasicSystemRequirements(); err != nil {
				return err
			}

			// Verificar dependencias SSL
			depErrors := utils.CheckSSLDependencies()
			if len(depErrors) > 0 {
				return fmt.Errorf("dependencias SSL faltantes:\n%s", utils.FormatDependencyErrors(depErrors))
			}

			// Configurar opciones
			if opts.Domain == "" {
				return fmt.Errorf("el dominio es obligatorio")
			}

			// Usar email de configuración si no se especifica
			if opts.Email == "" {
				if cfg.Email == "" {
					return fmt.Errorf("email requerido para SSL - configúralo en ~/.config/sitemanager/config.yaml o usa -e")
				}
				opts.Email = cfg.Email
			}
			
			// Verificar configuración SSL
			if err := cfg.ValidateSSLConfig(); err != nil {
				return fmt.Errorf("configuración SSL inválida: %v", err)
			}

			// Configurar usuario y directorios
			// Determinar si es un subdominio para configurar rutas correctas
			domainParts := strings.Split(opts.Domain, ".")
			if len(domainParts) > 2 && domainParts[0] != "www" {
				// Es un subdominio
				parentDomain := strings.Join(domainParts[1:], ".")
				opts.User = strings.Split(parentDomain, ".")[0]
				// Usar la nueva estructura: /home/dominio.com/subdominios/sub.dominio.com/
				parentHomeDir := filepath.Join("/home", parentDomain)
				opts.HomeDir = filepath.Join(parentHomeDir, "subdominios", opts.Domain)
			} else {
				// No es subdominio
				opts.User = domainParts[0]
				opts.HomeDir = filepath.Join("/home", opts.Domain)
			}

			// Verificar si el sitio existe
			if !siteExists(opts.Domain, cfg) {
				return fmt.Errorf("el sitio %s no existe, primero crea el sitio con 'sm site'", opts.Domain)
			}

			// Verificar si los certificados ya existen
			certPath := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", opts.Domain)
			if _, err := os.Stat(certPath); err == nil && !opts.Force {
				fmt.Printf("Los certificados SSL ya existen para %s. Use --force para regenerarlos.\n", opts.Domain)
			} else {
				// Obtener certificado SSL con Certbot
				if err := obtainSSLCertificate(&opts); err != nil {
					return err
				}
			}

			// Verificar si la configuración de Nginx ya tiene SSL
			nginxDir := filepath.Join(opts.HomeDir, "nginx")
			confFile := filepath.Join(nginxDir, fmt.Sprintf("%s.conf", opts.Domain))
			if currentConfig, err := os.ReadFile(confFile); err == nil {
				if strings.Contains(string(currentConfig), "ssl_certificate") && !opts.Force {
					fmt.Printf("La configuración de Nginx ya tiene SSL para %s. Use --force para actualizarla.\n", opts.Domain)
					return nil
				}
			}

			// Actualizar configuración de Nginx para usar SSL
			if err := updateNginxConfigWithSSL(&opts, cfg); err != nil {
				return err
			}

			// Recargar configuración de Nginx
			if err := reloadNginx(); err != nil {
				return err
			}

			fmt.Printf("SSL configurado correctamente para %s\n", opts.Domain)
			return nil
		},
	}

	// Agregar flags
	secureCmd.Flags().StringVarP(&opts.Domain, "domain", "d", "", "Dominio del sitio (obligatorio)")
	secureCmd.Flags().StringVarP(&opts.Email, "email", "e", "", "Email para Let's Encrypt (obligatorio)")
	secureCmd.Flags().BoolVar(&opts.Force, "force", false, "Forzar la regeneración de certificados SSL y actualización de configuración")

	// Marcar flags obligatorios
	secureCmd.MarkFlagRequired("domain")
	secureCmd.MarkFlagRequired("email")

	// Validación de requisitos antes de ejecutar
	secureCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Validar dominio
		if err := utils.ValidateDomain(opts.Domain); err != nil {
			return err
		}

		// Verificar requisitos
		return utils.CheckRequirements("secure", nil)
	}

	// Agregar comando al comando raíz
	rootCmd.AddCommand(secureCmd)
}

// siteExists verifica si el sitio existe
func siteExists(domain string, cfg *config.Config) bool {
	// Verificar si es un subdominio
	domainParts := strings.Split(domain, ".")
	var homeDir string
	var parentDomain string

	if len(domainParts) > 2 && domainParts[0] != "www" {
		// Es un subdominio, verificar si existe el dominio principal
		parentDomain = strings.Join(domainParts[1:], ".")
		parentConfFile := filepath.Join(cfg.SitesAvailable, fmt.Sprintf("%s.conf", parentDomain))
		if _, err := os.Stat(parentConfFile); os.IsNotExist(err) {
			fmt.Printf("El dominio principal %s no existe\n", parentDomain)
			return false
		}
		// Para subdominios, usar la nueva estructura: /home/dominio.com/subdominios/sub.dominio.com/
		parentHomeDir := filepath.Join("/home", parentDomain)
		homeDir = filepath.Join(parentHomeDir, "subdominios", domain)
	} else {
		homeDir = filepath.Join("/home", domain)
	}

	// Verificar si existe el directorio del sitio
	if _, err := os.Stat(homeDir); os.IsNotExist(err) {
		fmt.Printf("El directorio %s no existe\n", homeDir)
		return false
	}

	// Verificar si existe el archivo de configuración en sites-available
	confFile := filepath.Join(cfg.SitesAvailable, fmt.Sprintf("%s.conf", domain))
	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		fmt.Printf("El archivo de configuración %s no existe\n", confFile)
		return false
	}

	// Verificar si existe el directorio nginx
	nginxDir := filepath.Join(homeDir, "nginx")
	if _, err := os.Stat(nginxDir); os.IsNotExist(err) {
		fmt.Printf("El directorio nginx %s no existe\n", nginxDir)
		return false
	}

	// Verificar si existe el archivo de configuración en nginx
	nginxConfFile := filepath.Join(nginxDir, fmt.Sprintf("%s.conf", domain))
	if _, err := os.Stat(nginxConfFile); os.IsNotExist(err) {
		fmt.Printf("El archivo de configuración nginx %s no existe\n", nginxConfFile)
		return false
	}

	// Para subdominios, no es necesario verificar el directorio apps
	// ya que cada subdominio tiene su propia estructura

	return true
}

// obtainSSLCertificate obtiene un certificado SSL usando Certbot
func obtainSSLCertificate(opts *SecureOptions) error {
	fmt.Printf("Obteniendo certificado SSL para %s...\n", opts.Domain)

	// Verificar si Certbot está instalado
	if _, err := exec.LookPath("certbot"); err != nil {
		return fmt.Errorf("certbot no está instalado, instálalo primero")
	}

	// Crear y configurar el directorio para el desafío ACME
	webrootPath := filepath.Join(opts.HomeDir, "public_html")
	acmePath := filepath.Join(webrootPath, ".well-known", "acme-challenge")
	if err := os.MkdirAll(acmePath, 0755); err != nil {
		return fmt.Errorf("error al crear directorio para desafío ACME: %v", err)
	}

	// Asegurar que los permisos son correctos en toda la ruta
	if err := exec.Command("chmod", "-R", "755", webrootPath).Run(); err != nil {
		return fmt.Errorf("error al configurar permisos del webroot: %v", err)
	}

	// Asegurar que el propietario es correcto
	if err := exec.Command("chown", "-R", fmt.Sprintf("%s:www-data", opts.User), webrootPath).Run(); err != nil {
		return fmt.Errorf("error al configurar propietario del webroot: %v", err)
	}

	// Asegurar que el directorio padre tiene los permisos correctos para que www-data pueda acceder
	homeDir := filepath.Dir(webrootPath)
	if err := exec.Command("chmod", "755", homeDir).Run(); err != nil {
		return fmt.Errorf("error al configurar permisos del directorio padre: %v", err)
	}

	// Ejecutar Certbot en modo webroot
	cmd := exec.Command(
		"certbot", "certonly",
		"--webroot",
		"--webroot-path", webrootPath,
		"--email", opts.Email,
		"--domain", opts.Domain,
		"--agree-tos",
		"--non-interactive",
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al obtener certificado SSL: %v\n%s", err, output)
	}

	fmt.Printf("Certificado SSL obtenido correctamente para %s\n", opts.Domain)
	return nil
}

// updateNginxConfigWithSSL actualiza la configuración de Nginx para usar SSL
func updateNginxConfigWithSSL(opts *SecureOptions, cfg *config.Config) error {
	// Leer la plantilla SSL
	tmplPath := "ssl/ssl.conf.tmpl"
	tmplContent, err := utils.ReadTemplateFile(tmplPath)
	if err != nil {
		return err
	}

	// Crear plantilla
	tmpl, err := template.New("ssl").Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("error al parsear plantilla SSL: %v", err)
	}

	// Datos para la plantilla
	data := map[string]interface{}{
		"Domain":       opts.Domain,
		"CertPath":     fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", opts.Domain),
		"KeyPath":      fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", opts.Domain),
		"HomeDir":      opts.HomeDir,
		"RedirectHTTP": true,
		"PHP":          "8.4",
	}

	// Archivo de configuración actual
	nginxDir := filepath.Join(opts.HomeDir, "nginx")
	confFile := filepath.Join(nginxDir, fmt.Sprintf("%s.conf", opts.Domain))

	// Leer configuración actual
	currentConfig, err := os.ReadFile(confFile)
	if err != nil {
		return fmt.Errorf("error al leer configuración actual: %v", err)
	}

	// Verificar si ya tiene SSL configurado
	if strings.Contains(string(currentConfig), "ssl_certificate") {
		fmt.Println("La configuración ya tiene SSL, actualizando...")
	}

	// Crear archivo temporal para la nueva configuración
	tmpFile := filepath.Join(nginxDir, fmt.Sprintf("%s.conf.ssl", opts.Domain))
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("error al crear archivo temporal: %v", err)
	}
	defer file.Close()

	// Ejecutar plantilla
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("error al ejecutar plantilla SSL: %v", err)
	}

	// Cerrar archivo
	file.Close()

	// Reemplazar el archivo original
	if err := os.Rename(tmpFile, confFile); err != nil {
		return fmt.Errorf("error al reemplazar archivo: %v", err)
	}

	fmt.Printf("Configuración de Nginx actualizada con SSL para %s\n", opts.Domain)
	return nil
}

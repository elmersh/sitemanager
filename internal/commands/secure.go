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
			// Configurar opciones
			if opts.Domain == "" {
				return fmt.Errorf("el dominio es obligatorio")
			}

			if opts.Email == "" {
				return fmt.Errorf("el email es obligatorio para Let's Encrypt")
			}

			// Configurar usuario y directorios
			opts.User = strings.Split(opts.Domain, ".")[0]
			opts.HomeDir = filepath.Join("/home", opts.Domain)

			// Verificar si el sitio existe
			if !siteExists(opts.Domain, cfg) {
				return fmt.Errorf("el sitio %s no existe, primero crea el sitio con 'sm site'", opts.Domain)
			}

			// Obtener certificado SSL con Certbot
			if err := obtainSSLCertificate(&opts); err != nil {
				return err
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
	// Verificar si existe el archivo de configuración en sites-available
	confFile := filepath.Join(cfg.SitesAvailable, fmt.Sprintf("%s.conf", domain))
	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// obtainSSLCertificate obtiene un certificado SSL usando Certbot
func obtainSSLCertificate(opts *SecureOptions) error {
	fmt.Printf("Obteniendo certificado SSL para %s...\n", opts.Domain)

	// Verificar si Certbot está instalado
	if _, err := exec.LookPath("certbot"); err != nil {
		return fmt.Errorf("certbot no está instalado, instálalo primero")
	}

	// Ejecutar Certbot en modo webroot
	cmd := exec.Command(
		"certbot", "certonly",
		"--webroot",
		"--webroot-path", filepath.Join(opts.HomeDir, "public_html"),
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
	tmplPath := "templates/ssl/ssl.conf.tmpl"
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
	}

	// Archivo de configuración actual
	nginxDir := filepath.Join(opts.HomeDir, ".nginx")
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

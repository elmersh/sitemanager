package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// DependencyError representa un error de dependencia faltante
type DependencyError struct {
	Dependency   string
	Required     string
	InstallHint  string
	DocumentURL  string
}

func (e *DependencyError) Error() string {
	return fmt.Sprintf("dependencia faltante: %s - %s", e.Dependency, e.Required)
}

// CheckNginxDependency verifica si Nginx está instalado y funcionando
func CheckNginxDependency() error {
	if !commandExists("nginx") {
		return &DependencyError{
			Dependency:  "nginx",
			Required:    "requerido para crear y gestionar sitios web",
			InstallHint: "sudo apt install nginx",
			DocumentURL: "https://nginx.org/en/docs/install.html",
		}
	}
	
	// Verificar si nginx está corriendo
	if err := exec.Command("systemctl", "is-active", "nginx").Run(); err != nil {
		return &AppError{
			Type:    "service",
			Message: "nginx no está funcionando - ejecuta: sudo systemctl start nginx",
		}
	}
	
	return nil
}

// CheckPHPDependency verifica si PHP-FPM está instalado para sitios Laravel
func CheckPHPDependency(version string) error {
	if version == "" {
		version = "8.3" // Versión por defecto
	}
	
	phpFpm := fmt.Sprintf("php%s-fpm", version)
	
	if !commandExists("php") {
		return &DependencyError{
			Dependency:  "php",
			Required:    fmt.Sprintf("requerido para sitios Laravel (versión %s)", version),
			InstallHint: fmt.Sprintf("sudo apt install php%s-fpm php%s-mysql php%s-xml php%s-curl", version, version, version, version),
			DocumentURL: "https://www.php.net/manual/en/install.php",
		}
	}
	
	// Verificar PHP-FPM específicamente
	if err := exec.Command("systemctl", "is-active", phpFpm).Run(); err != nil {
		return &DependencyError{
			Dependency:  phpFpm,
			Required:    fmt.Sprintf("servicio PHP-FPM para versión %s", version),
			InstallHint: fmt.Sprintf("sudo apt install php%s-fpm && sudo systemctl start php%s-fpm", version, version),
			DocumentURL: "https://www.php.net/manual/en/install.fpm.php",
		}
	}
	
	return nil
}

// CheckNodeJSDependency verifica si Node.js está instalado para sitios Node.js
func CheckNodeJSDependency() error {
	if !commandExists("node") {
		return &DependencyError{
			Dependency:  "node",
			Required:    "requerido para sitios Node.js",
			InstallHint: "curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash - && sudo apt install nodejs",
			DocumentURL: "https://nodejs.org/en/download/",
		}
	}
	
	return nil
}

// CheckPM2Dependency verifica si PM2 está instalado para gestión de procesos Node.js
func CheckPM2Dependency() error {
	if !commandExists("pm2") {
		return &DependencyError{
			Dependency:  "pm2",
			Required:    "requerido para gestión de procesos Node.js",
			InstallHint: "sudo npm install -g pm2",
			DocumentURL: "https://pm2.keymetrics.io/docs/usage/quick-start/",
		}
	}
	
	return nil
}

// CheckComposerDependency verifica si Composer está instalado para proyectos Laravel
func CheckComposerDependency() error {
	if !commandExists("composer") {
		return &DependencyError{
			Dependency:  "composer",
			Required:    "requerido para gestión de dependencias Laravel",
			InstallHint: "curl -sS https://getcomposer.org/installer | php && sudo mv composer.phar /usr/local/bin/composer",
			DocumentURL: "https://getcomposer.org/doc/00-intro.md",
		}
	}
	
	return nil
}

// CheckCertbotDependency verifica si Certbot está instalado para SSL
func CheckCertbotDependency() error {
	if !commandExists("certbot") {
		return &DependencyError{
			Dependency:  "certbot",
			Required:    "requerido para certificados SSL automáticos",
			InstallHint: "sudo apt install certbot python3-certbot-nginx",
			DocumentURL: "https://certbot.eff.org/instructions",
		}
	}
	
	return nil
}

// CheckSiteTypeDependencies verifica todas las dependencias para un tipo de sitio específico
func CheckSiteTypeDependencies(siteType string, phpVersion string) []error {
	var errors []error
	
	// Nginx es obligatorio para todos los tipos de sitio
	if err := CheckNginxDependency(); err != nil {
		errors = append(errors, err)
	}
	
	switch siteType {
	case "laravel":
		if err := CheckPHPDependency(phpVersion); err != nil {
			errors = append(errors, err)
		}
		if err := CheckComposerDependency(); err != nil {
			errors = append(errors, err)
		}
		
	case "nodejs":
		if err := CheckNodeJSDependency(); err != nil {
			errors = append(errors, err)
		}
		if err := CheckPM2Dependency(); err != nil {
			errors = append(errors, err)
		}
		
	case "static":
		// Solo requiere nginx que ya se verificó arriba
		break
	}
	
	return errors
}

// CheckSSLDependencies verifica dependencias para SSL
func CheckSSLDependencies() []error {
	var errors []error
	
	if err := CheckNginxDependency(); err != nil {
		errors = append(errors, err)
	}
	
	if err := CheckCertbotDependency(); err != nil {
		errors = append(errors, err)
	}
	
	return errors
}

// FormatDependencyErrors formatea errores de dependencias para mostrar al usuario
func FormatDependencyErrors(errors []error) string {
	if len(errors) == 0 {
		return ""
	}
	
	var sb strings.Builder
	sb.WriteString("❌ Dependencias faltantes detectadas:\n\n")
	
	for i, err := range errors {
		if depErr, ok := err.(*DependencyError); ok {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, depErr.Dependency))
			sb.WriteString(fmt.Sprintf("   📋 %s\n", depErr.Required))
			sb.WriteString(fmt.Sprintf("   💡 Instalar: %s\n", depErr.InstallHint))
			if depErr.DocumentURL != "" {
				sb.WriteString(fmt.Sprintf("   📖 Docs: %s\n", depErr.DocumentURL))
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString(fmt.Sprintf("%d. %s\n\n", i+1, err.Error()))
		}
	}
	
	sb.WriteString("🔧 Instala las dependencias faltantes y vuelve a intentar.\n")
	return sb.String()
}

// WarnDependencyErrors muestra advertencias de dependencias sin detener la ejecución
func WarnDependencyErrors(errors []error) {
	if len(errors) == 0 {
		return
	}
	
	fmt.Print("⚠️  Advertencias de dependencias:\n\n")
	
	for i, err := range errors {
		if depErr, ok := err.(*DependencyError); ok {
			fmt.Printf("%d. %s - %s\n", i+1, depErr.Dependency, depErr.Required)
			fmt.Printf("   💡 %s\n\n", depErr.InstallHint)
		} else {
			fmt.Printf("%d. %s\n\n", i+1, err.Error())
		}
	}
}

// commandExists verifica si un comando existe en el sistema
func commandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// CheckBasicSystemRequirements verifica requisitos básicos del sistema
func CheckBasicSystemRequirements() error {
	// Verificar permisos de root
	if !CheckRoot() {
		return &AppError{
			Type:    "permission",
			Message: "se requieren permisos de root (sudo) para ejecutar este comando",
		}
	}
	
	// Verificar sistema operativo compatible
	if !IsLinux() {
		return &AppError{
			Type:    "system",
			Message: "SiteManager solo es compatible con sistemas Linux (Ubuntu/Debian)",
		}
	}
	
	return nil
}

// IsLinux verifica si el sistema operativo es Linux
func IsLinux() bool {
	return commandExists("apt") || commandExists("yum") || commandExists("systemctl")
}
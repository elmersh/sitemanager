// internal/utils/errors.go
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ErrorType representa el tipo de error
type ErrorType int

const (
	// ErrorGeneral es un error general
	ErrorGeneral ErrorType = iota
	// ErrorPermiso es un error de permisos
	ErrorPermiso
	// ErrorArchivo es un error relacionado con archivos
	ErrorArchivo
	// ErrorComando es un error de ejecución de comando
	ErrorComando
	// ErrorRed es un error de red
	ErrorRed
	// ErrorValidacion es un error de validación
	ErrorValidacion
)

// AppError representa un error de la aplicación
type AppError struct {
	Type    ErrorType
	Message string
	Cause   error
}

// Error implementa la interfaz error
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// NewError crea un nuevo error de aplicación
func NewError(errType ErrorType, message string, cause error) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// HandleError maneja un error y devuelve un mensaje apropiado
func HandleError(err error) string {
	if appErr, ok := err.(*AppError); ok {
		switch appErr.Type {
		case ErrorPermiso:
			return fmt.Sprintf("Error de permisos: %s. Asegúrate de estar ejecutando el comando con sudo.", appErr.Error())
		case ErrorArchivo:
			return fmt.Sprintf("Error de archivo: %s", appErr.Error())
		case ErrorComando:
			return fmt.Sprintf("Error al ejecutar comando: %s", appErr.Error())
		case ErrorRed:
			return fmt.Sprintf("Error de red: %s. Verifica tu conexión a internet.", appErr.Error())
		case ErrorValidacion:
			return fmt.Sprintf("Error de validación: %s", appErr.Error())
		default:
			return fmt.Sprintf("Error: %s", appErr.Error())
		}
	}
	return fmt.Sprintf("Error: %v", err)
}

// CheckRoot verifica si el usuario tiene permisos de root (sudo)
func CheckRoot() bool {
	if runtime.GOOS == "windows" {
		// En Windows, la verificación es diferente
		cmd := exec.Command("net", "session")
		if err := cmd.Run(); err != nil {
			return false
		}
		return true
	}

	return os.Geteuid() == 0
}

// CheckNginx verifica si Nginx está instalado y en ejecución
func CheckNginx() error {
	// Verificar si Nginx está instalado
	if _, err := exec.LookPath("nginx"); err != nil {
		return NewError(ErrorValidacion, "Nginx no está instalado", err)
	}

	// Verificar si Nginx está en ejecución
	cmd := exec.Command("systemctl", "is-active", "nginx")
	output, err := cmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "active" {
		return NewError(ErrorValidacion, "Nginx no está en ejecución", nil)
	}

	return nil
}

// CheckPM2 verifica si PM2 está instalado
func CheckPM2() error {
	if _, err := exec.LookPath("pm2"); err != nil {
		return NewError(ErrorValidacion, "PM2 no está instalado", err)
	}
	return nil
}

// CheckComposer verifica si Composer está instalado
func CheckComposer() error {
	if _, err := exec.LookPath("composer"); err != nil {
		return NewError(ErrorValidacion, "Composer no está instalado", err)
	}
	return nil
}

// CheckPHP verifica si PHP está instalado
func CheckPHP(version string) error {
	cmd := exec.Command("php", "-v")
	if output, err := cmd.CombinedOutput(); err != nil {
		return NewError(ErrorValidacion, "PHP no está instalado", err)
	} else if version != "" {
		if !strings.Contains(string(output), version) {
			return NewError(ErrorValidacion, fmt.Sprintf("PHP %s no está instalado", version), nil)
		}
	}
	return nil
}

// ValidateDomain valida un nombre de dominio
func ValidateDomain(domain string) error {
	if domain == "" {
		return NewError(ErrorValidacion, "El dominio no puede estar vacío", nil)
	}

	// Verificar formato del dominio
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return NewError(ErrorValidacion, "Formato de dominio inválido", nil)
	}

	// Verificar caracteres válidos
	for _, part := range parts {
		if len(part) == 0 {
			return NewError(ErrorValidacion, "El dominio contiene partes vacías", nil)
		}
		for _, c := range part {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
				return NewError(ErrorValidacion, "El dominio contiene caracteres inválidos", nil)
			}
		}
		if part[0] == '-' || part[len(part)-1] == '-' {
			return NewError(ErrorValidacion, "El dominio no puede comenzar o terminar con guion", nil)
		}
	}

	return nil
}

// ValidateRepository valida una URL de repositorio
func ValidateRepository(repo string, ssh bool) error {
	if repo == "" {
		return NewError(ErrorValidacion, "El repositorio no puede estar vacío", nil)
	}

	if ssh {
		// Validar formato SSH
		if !strings.HasPrefix(repo, "git@") || !strings.Contains(repo, ":") {
			return NewError(ErrorValidacion, "Formato de URL SSH inválido, debe ser git@github.com:usuario/repo.git", nil)
		}
	} else {
		// Validar formato HTTPS
		if !strings.HasPrefix(repo, "https://") {
			return NewError(ErrorValidacion, "Formato de URL HTTPS inválido, debe comenzar con https://", nil)
		}
	}

	return nil
}

// CheckRequirements verifica los requisitos para el comando
func CheckRequirements(command string, opts map[string]string) error {
	// Verificar permisos de root
	if !CheckRoot() {
		return NewError(ErrorPermiso, "Este comando debe ser ejecutado con sudo", nil)
	}

	// Verificar requisitos según el comando
	switch command {
	case "site":
		if err := CheckNginx(); err != nil {
			return err
		}
		if template, ok := opts["template"]; ok && template == "laravel" {
			if php, ok := opts["php"]; ok {
				if err := CheckPHP(php); err != nil {
					return err
				}
			}
		}
	case "secure":
		if err := CheckNginx(); err != nil {
			return err
		}
		// Verificar Certbot
		if _, err := exec.LookPath("certbot"); err != nil {
			return NewError(ErrorValidacion, "Certbot no está instalado", err)
		}
	case "deploy":
		if template, ok := opts["template"]; ok {
			if template == "laravel" {
				if err := CheckComposer(); err != nil {
					return err
				}
				if php, ok := opts["php"]; ok {
					if err := CheckPHP(php); err != nil {
						return err
					}
				}
			} else if template == "nodejs" {
				if err := CheckPM2(); err != nil {
					return err
				}
			}
		}
		// Verificar Git
		if _, err := exec.LookPath("git"); err != nil {
			return NewError(ErrorValidacion, "Git no está instalado", err)
		}
	}

	return nil
}

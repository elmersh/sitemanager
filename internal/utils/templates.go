// internal/utils/templates.go
package utils

import (
	"fmt"
	"os"

	"github.com/elmersh/sitemanager/internal/templates"
)

// ReadTemplateFile lee el contenido de un archivo de plantilla
func ReadTemplateFile(tmplPath string) (string, error) {
	// Primero intentar leer desde las plantillas embebidas
	content, err := templates.GetTemplateContent(tmplPath)
	if err == nil {
		return content, nil
	}

	// Si no se encuentra en las plantillas embebidas, buscar en rutas del sistema
	// Esta parte se mantiene para compatibilidad con versiones anteriores
	// o para permitir personalización de plantillas
	for _, dir := range GetTemplateDirs() {
		fullPath := dir + "/" + tmplPath
		if PathExists(fullPath) {
			fileContent, err := os.ReadFile(fullPath)
			if err == nil {
				return string(fileContent), nil
			}
		}
	}

	return "", fmt.Errorf("plantilla %s no encontrada", tmplPath)
}

// GetTemplateDirs devuelve una lista de directorios donde buscar plantillas
func GetTemplateDirs() []string {
	// Obtener directorio home del usuario
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/root"
	}

	return []string{
		"./templates",
		homeDir + "/.config/sitemanager/templates",
		"/etc/sitemanager/templates",
		"/usr/local/share/sitemanager/templates",
		"/usr/share/sitemanager/templates",
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/elmersh/sitemanager/internal/commands"
	"github.com/elmersh/sitemanager/internal/config"
	"github.com/spf13/cobra"
)

func main() {
	// Verificar si el usuario tiene permisos sudo
	if os.Geteuid() != 0 {
		fmt.Println("Este comando debe ser ejecutado con sudo")
		os.Exit(1)
	}

	// Cargar configuración
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error al cargar la configuración: %v\n", err)
		os.Exit(1)
	}

	// Comando raíz
	rootCmd := &cobra.Command{
		Use:   "sm",
		Short: "SiteManager - Herramienta para gestionar sitios web en VPS",
		Long: `SiteManager (sm) es una herramienta para gestionar rápidamente sitios web
en un servidor VPS, incluyendo configuraciones de Nginx, usuarios y
despliegue de aplicaciones como Laravel y Node.js.`,
	}

	// Agregar comandos
	commands.AddSiteCommand(rootCmd, cfg)
	commands.AddSecureCommand(rootCmd, cfg)
	commands.AddDeployCommand(rootCmd, cfg)

	// Ejecutar comando raíz
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

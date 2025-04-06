// cmd/sm/main.go actualizado
package main

import (
	"fmt"
	"os"

	"github.com/elmersh/sitemanager/internal/commands"
	"github.com/elmersh/sitemanager/internal/config"
	"github.com/elmersh/sitemanager/internal/utils"
	"github.com/spf13/cobra"
)

var version = "1.0.0"

func main() {
	// Verificar si el usuario tiene permisos sudo
	if !utils.CheckRoot() {
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
		Use:     "sm",
		Short:   "SiteManager - Herramienta para gestionar sitios web en VPS",
		Version: version,
		Long: `SiteManager (sm) es una herramienta para gestionar rápidamente sitios web
en un servidor VPS, incluyendo configuraciones de Nginx, usuarios y
despliegue de aplicaciones como Laravel y Node.js.`,
	}

	// Agregar comandos
	commands.AddSiteCommand(rootCmd, cfg)
	commands.AddSecureCommand(rootCmd, cfg)
	commands.AddDeployCommand(rootCmd, cfg)

	// Comando para verificar el estado del sistema
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Mostrar el estado del sistema",
		Run: func(cmd *cobra.Command, args []string) {
			// Verificar Nginx
			fmt.Print("Verificando Nginx... ")
			if err := utils.CheckNginx(); err != nil {
				fmt.Println("❌ No disponible")
				fmt.Println("  " + utils.HandleError(err))
			} else {
				fmt.Println("✅ Funcionando")
			}

			// Verificar PHP
			fmt.Print("Verificando PHP... ")
			if err := utils.CheckPHP(""); err != nil {
				fmt.Println("❌ No disponible")
				fmt.Println("  " + utils.HandleError(err))
			} else {
				fmt.Println("✅ Instalado")
			}

			// Verificar Certbot
			fmt.Print("Verificando Certbot... ")
			if _, err := os.Stat("/usr/bin/certbot"); os.IsNotExist(err) {
				fmt.Println("❌ No disponible")
				fmt.Println("  Certbot no está instalado")
			} else {
				fmt.Println("✅ Instalado")
			}

			// Verificar PM2
			fmt.Print("Verificando PM2... ")
			if err := utils.CheckPM2(); err != nil {
				fmt.Println("❌ No disponible")
				fmt.Println("  " + utils.HandleError(err))
			} else {
				fmt.Println("✅ Instalado")
			}

			// Verificar Composer
			fmt.Print("Verificando Composer... ")
			if err := utils.CheckComposer(); err != nil {
				fmt.Println("❌ No disponible")
				fmt.Println("  " + utils.HandleError(err))
			} else {
				fmt.Println("✅ Instalado")
			}

			// Obtener información del sistema
			sysInfo, err := utils.GetSystemInfo()
			if err == nil {
				fmt.Printf("\nInformación del sistema:\n")
				fmt.Printf("Hostname: %s\n", sysInfo.Hostname)
				fmt.Printf("Memoria disponible: %d MB\n", sysInfo.Memory)
				fmt.Printf("Dirección IP: %s\n", sysInfo.IPAddress)
			}
		},
	}

	rootCmd.AddCommand(statusCmd)

	// Ejecutar comando raíz
	if err := rootCmd.Execute(); err != nil {
		if appErr, ok := err.(*utils.AppError); ok {
			fmt.Println(utils.HandleError(appErr))
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}

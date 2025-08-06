# Documentación de SiteManager

Esta carpeta contiene documentación adicional para desarrolladores y contribuidores.

## Contenido

- [`BUILD.md`](BUILD.md) - Guía completa de compilación y empaquetado
- [`CLAUDE.md`](CLAUDE.md) - Guía para asistentes de IA que trabajen con el proyecto

## Para Usuarios

La documentación principal para usuarios está en el [README.md](../README.md) principal.

## Para Desarrolladores

Si eres desarrollador y quieres contribuir al proyecto:

1. Lee el [CONTRIBUTING.md](../CONTRIBUTING.md) para las guías de contribución
2. Consulta [`BUILD.md`](BUILD.md) para instrucciones de compilación
3. Revisa [`CLAUDE.md`](CLAUDE.md) si trabajas con asistentes de IA

## Estructura de Comandos

```
sm
├── status          # Verificar estado del sistema
├── site            # Crear/configurar sitios
├── secure          # Configurar SSL/HTTPS  
├── deploy          # Desplegar aplicaciones
├── env             # Gestionar variables de entorno
├── self-update     # Actualizar SiteManager
└── version         # Información de versión
```

## Arquitectura

```
internal/
├── commands/       # Implementación de comandos CLI
├── config/         # Sistema de configuración
├── templates/      # Templates para archivos de config
└── utils/          # Utilidades compartidas
```

# Changelog

Todos los cambios notables de este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/),
y este proyecto adhiere al [Versionado Semántico](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-08-05

### Release inicial

- Primera versión pública de SiteManager
- Soporte para sitios Laravel, Node.js y estáticos
- Configuración automática de Nginx con templates optimizados
- Integración completa con SSL/TLS via Certbot
- Sistema de plantillas para configuraciones personalizables
- Comando `sm site` para crear y configurar sitios web
- Comando `sm secure` para configurar certificados SSL
- Comando `sm deploy` para desplegar aplicaciones desde Git
- Comando `sm env` para gestionar variables de entorno
- Comando `sm self-update` para actualizaciones automáticas
- Detección automática de frameworks (NestJS, NextJS, Express, etc.)
- Soporte para subdominios con configuración automática
- Gestión de usuarios del sistema y permisos
- Scripts de instalación y desinstalación automáticos
- Documentación completa en español
- Templates para issues y contribuciones en GitHub

### Instalación

```bash
# Instalación rápida
curl -fsSL https://raw.githubusercontent.com/elmersh/sitemanager/main/install.sh | sudo bash

# Auto-actualización (si ya está instalado)
sudo sm self-update
```

### Documentación

- [Guía de instalación](docs/BUILD.md)
- [Documentación completa](README.md)
- [Contribuir](CONTRIBUTING.md)
- [Reportar issues](https://github.com/elmersh/sitemanager/issues)

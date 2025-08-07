# Makefile para sitemanager

# Variables
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
SHAREDIR ?= $(PREFIX)/share/sitemanager
CONFDIR ?= /etc/sitemanager

# GO variables
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get

# Binary name
BINARY_NAME = sm

# Build flags
VERSION ?= 0.2.2
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Ubuntu build variables
UBUNTU_GOOS = linux
UBUNTU_GOARCH = amd64
DIST_DIR = dist
PACKAGE_NAME = sitemanager-0.2.2
UBUNTU_PACKAGE = $(DIST_DIR)/$(PACKAGE_NAME)-linux-amd64.tar.gz

.PHONY: all build clean test deps install uninstall ubuntu release changelog

all: build

deps:
	@echo "Instalando dependencias..."
	$(GOGET) github.com/spf13/cobra@latest
	$(GOGET) gopkg.in/yaml.v3@latest

build: deps
	@echo "Compilando sitemanager..."
	cd cmd/sm && $(GOBUILD) $(LDFLAGS) -o ../../$(BINARY_NAME)

test:
	@echo "Ejecutando tests..."
	$(GOTEST) -v ./...

clean:
	@echo "Limpiando..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(DIST_DIR)
	rm -rf sitemanager

install: build
	@echo "Instalando sitemanager..."
	mkdir -p $(BINDIR)
	mkdir -p $(SHAREDIR)/templates/nginx
	mkdir -p $(SHAREDIR)/templates/ssl
	mkdir -p $(CONFDIR)
	cp $(BINARY_NAME) $(BINDIR)
	cp templates/nginx/*.tmpl $(SHAREDIR)/templates/nginx/
	cp templates/ssl/*.tmpl $(SHAREDIR)/templates/ssl/
	@echo "SiteManager instalado en $(BINDIR)/$(BINARY_NAME)"

uninstall:
	@echo "Desinstalando sitemanager..."
	rm -f $(BINDIR)/$(BINARY_NAME)
	rm -rf $(SHAREDIR)
	@echo "SiteManager desinstalado"

ubuntu: clean deps
	@echo "Compilando para Ubuntu..."
	# Crear estructura de directorios en dist
	mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)/bin
	mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)/templates/nginx
	mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)/templates/ssl
	mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)/scripts
	
	# Compilar binario para Ubuntu
	cd cmd/sm && GOOS=$(UBUNTU_GOOS) GOARCH=$(UBUNTU_GOARCH) $(GOBUILD) $(LDFLAGS) -o ../../$(DIST_DIR)/$(PACKAGE_NAME)/bin/$(BINARY_NAME)
	
	# Copiar templates
	cp internal/templates/nginx/*.tmpl $(DIST_DIR)/$(PACKAGE_NAME)/templates/nginx/
	cp internal/templates/ssl/*.tmpl $(DIST_DIR)/$(PACKAGE_NAME)/templates/ssl/
	
	# Crear script de instalación
	@echo "Creando script de instalación..."
	@printf '#!/bin/bash\nset -e\n\n# Verificar permisos de sudo\nif [ "$$EUID" -ne 0 ]; then\n    echo "❌ Este script debe ejecutarse con sudo"\n    exit 1\nfi\n\n# Variables\nPREFIX=$${PREFIX:-/usr/local}\nBINDIR=$$PREFIX/bin\nSHAREDIR=$$PREFIX/share/sitemanager\nCONFDIR=/etc/sitemanager\n\necho "🚀 Instalando SiteManager..."\n\n# Crear directorios\nmkdir -p $$BINDIR\nmkdir -p $$SHAREDIR/templates/nginx\nmkdir -p $$SHAREDIR/templates/ssl\nmkdir -p $$CONFDIR/skel\n\n# Instalar binario\ncp bin/$(BINARY_NAME) $$BINDIR/\nchmod +x $$BINDIR/$(BINARY_NAME)\n\n# Instalar templates\ncp templates/nginx/*.tmpl $$SHAREDIR/templates/nginx/\ncp templates/ssl/*.tmpl $$SHAREDIR/templates/ssl/\n\n# Crear enlace simbólico si no existe\nif [ ! -L /usr/bin/$(BINARY_NAME) ]; then\n    ln -s $$BINDIR/$(BINARY_NAME) /usr/bin/$(BINARY_NAME)\nfi\n\necho "✅ SiteManager instalado correctamente"\necho "📖 Ejecuta '"'"'sudo $(BINARY_NAME) status'"'"' para verificar el sistema"\n' > $(DIST_DIR)/$(PACKAGE_NAME)/install.sh
	@chmod +x $(DIST_DIR)/$(PACKAGE_NAME)/install.sh
	
	# Crear script de desinstalación
	@echo "Creando script de desinstalación..."
	@printf '#!/bin/bash\nset -e\n\n# Verificar permisos de sudo\nif [ "$$EUID" -ne 0 ]; then\n    echo "❌ Este script debe ejecutarse con sudo"\n    exit 1\nfi\n\n# Variables\nPREFIX=$${PREFIX:-/usr/local}\nBINDIR=$$PREFIX/bin\nSHAREDIR=$$PREFIX/share/sitemanager\nCONFDIR=/etc/sitemanager\n\necho "🗑️  Desinstalando SiteManager..."\n\n# Eliminar binario y enlace simbólico\nrm -f $$BINDIR/$(BINARY_NAME)\nrm -f /usr/bin/$(BINARY_NAME)\n\n# Eliminar templates\nrm -rf $$SHAREDIR\n\n# Preguntar si eliminar configuración\nread -p "¿Eliminar configuración del sistema? (y/N): " -n 1 -r\necho\nif [[ $$REPLY =~ ^[Yy]$$ ]]; then\n    rm -rf $$CONFDIR\n    echo "📁 Configuración eliminada"\nfi\n\necho "✅ SiteManager desinstalado correctamente"\n' > $(DIST_DIR)/$(PACKAGE_NAME)/uninstall.sh
	@chmod +x $(DIST_DIR)/$(PACKAGE_NAME)/uninstall.sh
	
	# Crear README para la distribución
	@echo "Creando README para distribución..."
	@printf '# SiteManager v0.2.2 - Distribución Ubuntu/Debian\n\nEste paquete contiene SiteManager compilado para sistemas Ubuntu/Debian (Linux AMD64).\n\n## Instalación\n\n```bash\n# Extraer el paquete\ntar -xzf sitemanager-0.2.2-linux-amd64.tar.gz\ncd sitemanager-0.2.2/\n\n# Instalar (requiere sudo)\nsudo ./install.sh\n```\n\n## Verificación\n\n```bash\n# Verificar instalación\nsudo sm status\n\n# Ver versión\nsm --version\n\n# Ver ayuda\nsm --help\n```\n\n## Desinstalación\n\n```bash\nsudo ./uninstall.sh\n```\n\n## Contenido del Paquete\n\n- `bin/sm` - Binario principal\n- `templates/` - Plantillas de configuración Nginx y SSL\n- `install.sh` - Script de instalación\n- `uninstall.sh` - Script de desinstalación\n- `README.md` - Este archivo\n\n## Requisitos del Sistema\n\n- Ubuntu 18.04+ o Debian 9+\n- Nginx\n- PHP-FPM (para sitios Laravel)\n- Node.js y PM2 (para sitios Node.js)\n- Certbot (para SSL)\n\n## Soporte\n\nPara más información: https://github.com/elmersh/sitemanager\n' > $(DIST_DIR)/$(PACKAGE_NAME)/README.md
	
	# Crear el paquete tar.gz
	@echo "Creando paquete comprimido..."
	cd $(DIST_DIR) && tar -czf $(PACKAGE_NAME)-linux-amd64.tar.gz $(PACKAGE_NAME)/
	
	@echo "✅ Paquete Ubuntu creado exitosamente:"
	@echo "   📁 Estructura: $(DIST_DIR)/$(PACKAGE_NAME)/"
	@echo "   📦 Paquete: $(UBUNTU_PACKAGE)"
	@echo ""
	@echo "Para instalar en Ubuntu/Debian:"
	@echo "   tar -xzf $(UBUNTU_PACKAGE)"
	@echo "   cd $(PACKAGE_NAME)/"
	@echo "   sudo ./install.sh"

# Comando para crear un release
release:
	@echo "🚀 Creando release v$(VERSION)..."
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Error: VERSION no está definido"; \
		echo "   Uso: make release VERSION=1.2.0"; \
		exit 1; \
	fi
	
	# Verificar que estemos en una rama limpia
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "❌ Error: Hay cambios sin commit"; \
		echo "   Commit tus cambios antes de crear un release"; \
		exit 1; \
	fi
	
	# Actualizar versión en el Makefile
	@sed -i.bak 's/VERSION ?= .*/VERSION ?= $(VERSION)/' Makefile
	@rm -f Makefile.bak
	
	# Generar changelog
	@echo "📝 Generando changelog..."
	@$(MAKE) changelog VERSION=$(VERSION)
	
	# Commit cambios de versión
	@git add Makefile CHANGELOG.md
	@git commit -m "Release v$(VERSION)" || true
	
	# Crear tag
	@echo "🏷️  Creando tag v$(VERSION)..."
	@git tag -a "v$(VERSION)" -m "Release v$(VERSION)"
	
	@echo ""
	@echo "✅ Release v$(VERSION) creado exitosamente"
	@echo ""
	@echo "📋 Próximos pasos:"
	@echo "   1. Revisar el changelog: cat CHANGELOG.md"
	@echo "   2. Subir cambios: git push origin main"
	@echo "   3. Subir tag: git push origin v$(VERSION)"
	@echo "   4. GitHub Actions compilará automáticamente y creará la release"
	@echo ""

# Comando para generar changelog
changelog:
	@echo "📝 Generando changelog para v$(VERSION)..."
	@echo "# Changelog" > CHANGELOG.md
	@echo "" >> CHANGELOG.md
	
	# Obtener el tag anterior si existe
	@PREV_TAG=$$(git describe --tags --abbrev=0 v$(VERSION)^ 2>/dev/null || echo ""); \
	if [ -n "$$PREV_TAG" ]; then \
		echo "## [$(VERSION)] - $$(date +%Y-%m-%d)" >> CHANGELOG.md; \
		echo "" >> CHANGELOG.md; \
		echo "### Cambios desde $$PREV_TAG" >> CHANGELOG.md; \
		echo "" >> CHANGELOG.md; \
		git log --oneline --no-merges $$PREV_TAG..HEAD | sed 's/^/- /' >> CHANGELOG.md; \
	else \
		echo "## [$(VERSION)] - $$(date +%Y-%m-%d)" >> CHANGELOG.md; \
		echo "" >> CHANGELOG.md; \
		echo "### Release inicial" >> CHANGELOG.md; \
		echo "" >> CHANGELOG.md; \
		echo "- Primera versión de SiteManager" >> CHANGELOG.md; \
		echo "- Soporte para sitios Laravel, Node.js y estáticos" >> CHANGELOG.md; \
		echo "- Configuración automática de Nginx" >> CHANGELOG.md; \
		echo "- Integración con SSL/TLS via Certbot" >> CHANGELOG.md; \
		echo "- Sistema de plantillas para configuraciones" >> CHANGELOG.md; \
	fi
	
	@echo "" >> CHANGELOG.md
	@echo "### Instalación" >> CHANGELOG.md
	@echo "" >> CHANGELOG.md
	@echo '```bash' >> CHANGELOG.md
	@echo "# Instalación rápida" >> CHANGELOG.md
	@echo 'curl -fsSL https://raw.githubusercontent.com/elmersh/sitemanager/main/install.sh | sudo bash' >> CHANGELOG.md
	@echo "" >> CHANGELOG.md
	@echo "# Auto-actualización (si ya está instalado)" >> CHANGELOG.md
	@echo "sudo sm self-update" >> CHANGELOG.md
	@echo '```' >> CHANGELOG.md
	@echo "" >> CHANGELOG.md
	@echo "### Documentación" >> CHANGELOG.md
	@echo "" >> CHANGELOG.md
	@echo "- [Guía de instalación](BUILD.md)" >> CHANGELOG.md
	@echo "- [Documentación completa](README.md)" >> CHANGELOG.md
	@echo "- [Reportar issues](https://github.com/elmersh/sitemanager/issues)" >> CHANGELOG.md
	
	@echo "✅ Changelog generado: CHANGELOG.md"
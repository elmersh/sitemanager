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
LDFLAGS = -ldflags "-X main.Version=1.0.0"

# Ubuntu build variables
UBUNTU_BINARY = sitemanager/sm
UBUNTU_PACKAGE = sitemanager.tar.gz
UBUNTU_GOOS = linux
UBUNTU_GOARCH = amd64

.PHONY: all build clean test deps install uninstall ubuntu

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
	rm -f $(UBUNTU_PACKAGE)
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
	mkdir -p sitemanager
	cd cmd/sm && GOOS=$(UBUNTU_GOOS) GOARCH=$(UBUNTU_GOARCH) $(GOBUILD) $(LDFLAGS) -o ../../$(UBUNTU_BINARY)
	@echo "Creando script de instalación..."
	@echo '#!/bin/bash' > sitemanager/install.sh
	@echo 'if [ -f /usr/local/bin/sm ]; then' >> sitemanager/install.sh
	@echo '    rm /usr/local/bin/sm' >> sitemanager/install.sh
	@echo 'fi' >> sitemanager/install.sh
	@echo 'cp sm /usr/local/bin/' >> sitemanager/install.sh
	@echo 'chmod +x /usr/local/bin/sm' >> sitemanager/install.sh
	@echo 'echo "SiteManager instalado en /usr/local/bin/sm"' >> sitemanager/install.sh
	chmod +x sitemanager/install.sh
	@echo "Creando paquete..."
	tar -czf $(UBUNTU_PACKAGE) sitemanager
	@echo "Paquete creado: $(UBUNTU_PACKAGE)"
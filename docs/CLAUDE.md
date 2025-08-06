# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SiteManager (sm) is a CLI tool for rapidly managing websites on a VPS server. It automates the configuration of Nginx, users, SSL certificates, and deployment of Laravel and Node.js applications. The tool is written in Go using the Cobra CLI framework.

## Build and Development Commands

### Building the application
```bash
# Build the binary
make build

# Build with dependencies
make all

# Clean build artifacts
make clean
```

### Testing
```bash
# Run tests
make test
```

### Installation
```bash
# Install locally (copies binary to /usr/local/bin and templates to /usr/local/share/sitemanager)
make install

# Uninstall
make uninstall

# Build Ubuntu package
make ubuntu
```

### Running the application
```bash
# The binary is called 'sm' and requires sudo privileges
sudo ./sm status
sudo ./sm site -d example.com -t laravel
sudo ./sm site -d static-site.com -t static
sudo ./sm secure -d example.com -e admin@example.com
sudo ./sm deploy -d example.com -r https://github.com/user/repo.git
```

## Architecture Overview

### Core Structure
- **Entry point**: `cmd/sm/main.go` - Main application with Cobra root command
- **Commands**: `internal/commands/` - Individual command implementations (site, secure, deploy, env)
- **Configuration**: `internal/config/` - YAML-based configuration management
- **Utilities**: `internal/utils/` - Common functions for system checks, error handling, and operations
- **Templates**: `internal/templates/` - Nginx configuration templates for different site types

### Key Components

1. **Command System**: Uses Cobra for CLI interface with subcommands for different operations
2. **Configuration Management**: YAML configuration file at `~/.config/sitemanager.yaml` with sensible defaults
3. **Template System**: Go text templates for generating Nginx configurations
4. **User Management**: Creates system users and directories for each domain
5. **SSL Integration**: Automated SSL certificate generation via Certbot
6. **Framework Detection**: Automatic detection of Node.js frameworks (NestJS, NextJS, Express, etc.)

### Site Types and Templates
- **Laravel**: PHP-FPM integration with version selection, includes composer and migration support
- **Node.js**: PM2 process management with port allocation, supports multiple frameworks
- **Static**: HTML/CSS/JS sites with optimized Nginx configuration, automatic file structure generation
- **Subdomain Support**: Automatic detection and configuration for subdomains

### Database Integration
- Supports PostgreSQL, MySQL, and MongoDB
- Automatic database creation and user setup
- Prisma ORM detection and migration support

## Important Configuration Details

### Default Paths
- Nginx config: `/etc/nginx`
- Sites available: `/etc/nginx/sites-available`
- Sites enabled: `/etc/nginx/sites-enabled`
- User home directories: `/home/{domain}`
- Skeleton directory: `/etc/sitemanager/skel`

### Required System Dependencies
- Nginx
- PHP-FPM (for Laravel sites)
- Node.js and PM2 (for Node.js sites)
- Certbot (for SSL)
- Composer (for Laravel)
- PostgreSQL/MySQL (optional)

### Permission Requirements
The application requires sudo privileges as it:
- Creates system users
- Modifies Nginx configurations
- Manages system services
- Creates directories in protected locations

## Development Guidelines

### Error Handling
- Uses custom `AppError` type in `utils/errors.go`
- Consistent error messages in Spanish (application is Spanish-focused)
- System validation checks before operations

### Template Management
- Templates are embedded and installed during `make install`
- Separate templates for domains vs subdomains
- Dynamic variable substitution for domain-specific configurations

### Testing Approach
- System integration focused (requires actual system services)
- Use `sudo sm status` to verify system dependencies before testing
- Test with actual domain configurations in development

### Adding New Features
1. Add command in `internal/commands/`
2. Register command in `cmd/sm/main.go`
3. Add any new templates to `internal/templates/`
4. Update configuration struct if needed
5. Add utility functions to appropriate `internal/utils/` file

## Deployment and Distribution

The application is distributed as a single binary with embedded templates. The Makefile handles cross-compilation for Ubuntu servers and creates deployment packages.
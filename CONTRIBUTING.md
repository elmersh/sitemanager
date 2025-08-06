# Contribuir a SiteManager

¡Gracias por tu interés en contribuir a SiteManager! Esta guía te ayudará a comenzar.

## Código de Conducta

Al participar en este proyecto, te comprometes a mantener un entorno amigable y acogedor para todos.

## Cómo Contribuir

### Reportar Bugs

1. Verifica que el bug no haya sido reportado ya en [Issues](https://github.com/elmersh/sitemanager/issues)
2. Abre un nuevo issue con:
   - Descripción clara del problema
   - Pasos para reproducir el bug
   - Comportamiento esperado vs actual
   - Información del sistema (Ubuntu/Debian version, Go version)

### Sugerir Funcionalidades

1. Abre un issue con la etiqueta "enhancement"
2. Describe claramente la funcionalidad propuesta
3. Explica por qué sería útil para otros usuarios

### Contribuir Código

1. Haz fork del repositorio
2. Crea una rama para tu funcionalidad:
   ```bash
   git checkout -b feature/mi-nueva-funcionalidad
   ```
3. Realiza tus cambios siguiendo las guías de estilo
4. Añade tests si es necesario
5. Asegúrate de que todos los tests pasen:
   ```bash
   make test
   ```
6. Confirma que el código compila correctamente:
   ```bash
   make build
   ```
7. Haz commit con un mensaje descriptivo
8. Push a tu fork y crea un Pull Request

## Configuración del Entorno de Desarrollo

### Requisitos

- Go 1.23.0 o superior
- Make
- Git

### Configuración

```bash
# Clonar tu fork
git clone https://github.com/TU_USUARIO/sitemanager.git
cd sitemanager

# Instalar dependencias
go mod download

# Compilar
make build

# Ejecutar tests
make test
```

## Guías de Estilo

### Código Go

- Usa `gofmt` para formatear el código
- Sigue las convenciones estándar de Go
- Añade comentarios para funciones públicas
- Usa nombres descriptivos para variables y funciones

### Commits

- Usa mensajes de commit claros y descriptivos
- Primera línea: resumen breve (50 caracteres máximo)
- Líneas adicionales: descripción detallada si es necesario

Ejemplo:
```
feat: añadir soporte para MongoDB

- Implementar configuración automática de MongoDB
- Añadir templates para conexiones de base de datos
- Actualizar documentación con ejemplos de uso
```

### Pull Requests

- Título claro que describa el cambio
- Descripción detallada de qué hace el PR
- Referencia a issues relacionados
- Screenshots si incluye cambios visuales

## Estructura del Proyecto

```
sitemanager/
├── cmd/sm/           # Punto de entrada de la aplicación
├── internal/
│   ├── commands/     # Comandos CLI
│   ├── config/       # Gestión de configuración
│   ├── templates/    # Templates de archivos de configuración
│   └── utils/        # Utilidades compartidas
├── docs/            # Documentación adicional
└── scripts/         # Scripts de desarrollo y construcción
```

## Testing

- Añade tests para nuevas funcionalidades
- Asegúrate de que los tests existentes sigan pasando
- Usa tests unitarios para lógica compleja
- Considera integration tests para comandos completos

## Documentación

- Actualiza el README.md si cambias funcionalidades
- Añade ejemplos de uso para nuevas funciones
- Documenta nuevas opciones de línea de comandos
- Usa comentarios en el código para lógica compleja

## Preguntas

Si tienes preguntas sobre cómo contribuir, no dudes en:

- Abrir un issue con la etiqueta "question"
- Contactar a los mantenedores

¡Gracias por contribuir a SiteManager!

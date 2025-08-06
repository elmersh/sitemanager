# Security Policy

## Supported Versions

Las siguientes versiones de SiteManager reciben actualizaciones de seguridad:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

Si encuentras una vulnerabilidad de seguridad en SiteManager, por favor repórtala de forma responsable:

### Proceso de Reporte

1. **NO crear un issue público** para vulnerabilidades de seguridad
2. Envía un email a: **[tu-email@dominio.com]** (reemplazar con email real)
3. Incluye la siguiente información:
   - Descripción detallada de la vulnerabilidad
   - Pasos para reproducir el problema
   - Versión afectada de SiteManager
   - Impacto potencial
   - Solución sugerida (si tienes alguna)

### Qué Esperar

- **Confirmación inicial**: Dentro de 48 horas
- **Evaluación inicial**: Dentro de 5 días laborables
- **Actualizaciones regulares**: Cada 7 días hasta la resolución
- **Tiempo de resolución**: Dependiendo de la severidad
  - Crítica: 1-7 días
  - Alta: 7-30 días
  - Media/Baja: 30-90 días

### Divulgación Responsable

- Trabajaremos contigo para entender y resolver el problema
- Te acreditaremos en el changelog si lo deseas
- Coordinaremos la divulgación pública después de la corrección
- No revelaremos tu información de contacto sin permiso

### Alcance

Esta política se aplica a:
- El binario principal de SiteManager
- Scripts de instalación y configuración
- Templates de configuración incluidos
- Documentación que pueda llevar a configuraciones inseguras

### Fuera del Alcance

- Vulnerabilidades en dependencias de terceros (repórtalas directamente a los mantenedores)
- Problemas de configuración del servidor no relacionados directamente con SiteManager
- Ataques de ingeniería social

## Mejores Prácticas de Seguridad

### Para Usuarios

1. **Mantén SiteManager actualizado**:
   ```bash
   sudo sm self-update
   ```

2. **Revisa los permisos de archivos**:
   - Los archivos de configuración deben tener permisos restrictivos
   - Las claves SSH deben estar protegidas (600)

3. **Monitorea los logs**:
   - Revisa regularmente los logs de Nginx
   - Monitorea intentos de acceso no autorizados

4. **Configuración SSL segura**:
   - Usa certificados válidos (no staging) en producción
   - Mantén Certbot actualizado

### Para Desarrolladores

1. **Validación de entrada**: Siempre valida y sanitiza la entrada del usuario
2. **Principio de menor privilegio**: Ejecuta solo las operaciones necesarias con privilegios elevados
3. **Logging seguro**: No registres información sensible en logs
4. **Manejo de secretos**: Nunca hardcodees contraseñas o tokens

## Historial de Vulnerabilidades

Actualmente no hay vulnerabilidades reportadas públicamente.

---

Agradecemos a la comunidad de seguridad por ayudar a mantener SiteManager seguro.

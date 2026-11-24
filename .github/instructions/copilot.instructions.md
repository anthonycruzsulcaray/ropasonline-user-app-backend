---
description: "Reglas generales para desarrollar, probar y revisar esta API REST de tienda de ropa en Go."
applyTo: "**"
---

# Instrucciones del proyecto

## Contexto

- Este repositorio contiene una API REST para una tienda en línea de ropa.
- El backend está escrito en Go y usa Gin. La versión de Go declarada en `go.mod` es la referencia; no la cambies sin necesidad y sin explicar el motivo.
- El proyecto usará exclusivamente Arquitectura Hexagonal (Puertos y Adaptadores), sin combinarla con otras arquitecturas. Separa el núcleo de negocio (`domain` y `application`) de los adaptadores externos (`infrastructure`), de modo que se puedan cambiar bases de datos o protocolos sin modificar la lógica central. No asumas que esta estructura ya existe: primero inspecciona el código actual.
- El proyecto usará MySQL y MongoDB de forma híbrida: MySQL será la fuente de verdad para operaciones transaccionales y MongoDB podrá respaldar APIs, consultas pesadas o lecturas optimizadas. Cuando un mismo dato se mantenga en ambas bases, define cómo se sincroniza y cómo se recupera ante fallos.
- Docker y Docker Compose se usarán para empaquetar y ejecutar la aplicación y sus dependencias. El despliegue objetivo es una instancia EC2 de AWS.

## Arquitectura hexagonal y diseño

- Mantén las dependencias dirigidas hacia el dominio: dominio y aplicación no deben depender de Gin, drivers de base de datos ni detalles de AWS o Docker.
- Usa el dominio para reglas e invariantes del negocio, como disponibilidad de inventario y condiciones de una orden. No pongas reglas de negocio en handlers HTTP ni en repositorios.
- Usa la aplicación para coordinar casos de uso y define allí los puertos que estos necesitan. Implementa los puertos de salida en adaptadores de infraestructura.
- Trata HTTP como un adaptador de entrada: sus handlers validan/transforman la solicitud, invocan un caso de uso y traducen su resultado a una respuesta HTTP.
- Introduce entidades, agregados, interfaces o capas solo cuando resuelvan una necesidad real. Evita abstracciones ceremoniales y refactors ajenos a la tarea.
- Sigue la organización y las convenciones que ya existan. Antes de crear archivos o mover código, revisa las implementaciones y pruebas cercanas.

## Go y API REST

- Escribe Go idiomático, con nombres claros, errores explícitos y funciones acotadas. Usa `context.Context` en operaciones de I/O y respeta cancelación y deadlines.
- Envuelve errores con contexto cuando se propaguen; no ocultes errores ni devuelvas detalles internos, consultas SQL o stack traces a clientes HTTP.
- Mantén separados los modelos de transporte HTTP y los modelos del dominio cuando sus responsabilidades o contratos difieran.
- Valida entradas en los límites del sistema y devuelve códigos HTTP coherentes. No confíes en que los datos recibidos del cliente sean válidos.
- No cambies contratos públicos, rutas, dependencias o esquemas de persistencia sin que la tarea lo requiera; describe cualquier cambio incompatible.

## Persistencia

- Usa MySQL para datos que requieren consistencia, como órdenes e inventario. Agrupa en una misma transacción los cambios que deban confirmarse o revertirse juntos.
- Ante cualquier error antes del commit, asegúrate de que la transacción haga rollback. No confirmes parcialmente una operación de negocio.
- Considera concurrencia al modificar inventario; una lectura seguida de una actualización sin bloqueo o condición atómica puede permitir sobreventa.
- Usa MongoDB cuando una API necesite búsquedas o consultas pesadas, filtros flexibles o un modelo de lectura optimizado, y justifica su uso según el caso. Define filtros, paginación e índices.
- Si un dato debe persistirse en MySQL y MongoDB, establece cuál es la fuente de verdad y una estrategia explícita de sincronización, consistencia y recuperación. No supongas que las escrituras en ambas bases son atómicas ni hagas escrituras duales sin manejar fallos, reintentos e idempotencia.
- Para mantener MongoDB sincronizado con cambios transaccionales de MySQL, considera el patrón Outbox u otro mecanismo durable adecuado. Documenta si la sincronización es inmediata o eventual y cómo se detectan/reparan datos desincronizados.
- Usa consultas parametrizadas y evita construir SQL concatenando entradas del usuario. Mantén credenciales fuera del código y del control de versiones.

## Configuración, contenedores y seguridad

- Guarda las credenciales y cadenas de conexión —incluidas las de MySQL y MongoDB— en un archivo `.env` local y léelas desde la aplicación mediante variables de entorno. No las hardcodees; valida la configuración obligatoria al iniciar y falla claramente si falta.
- Mantén `.env` excluido de Git y del contexto/imagen de Docker. Si agregas `.env.example`, incluye únicamente nombres de variables y valores de ejemplo no sensibles. En Docker Compose, declara explícitamente cómo se pasan las variables al contenedor; no asumas que el `.env` usado para interpolación se carga automáticamente en la aplicación.
- Mantén `Dockerfile`, `.dockerignore` y `docker-compose.yml` coherentes entre sí y con las necesidades reales del proyecto. No incluyas secretos, archivos locales ni artefactos innecesarios en la imagen.
- Para cambios de despliegue, considera señales de apagado, cierre de conexiones, health checks, persistencia de datos y exposición mínima de puertos.
- No abras MySQL o MongoDB a Internet ni incluyas claves, tokens, contraseñas o cadenas de conexión reales en archivos versionados, logs o ejemplos ejecutables. En EC2, conserva el `.env` fuera del repositorio y protégelo con permisos restrictivos.
- No añadas servicios, dependencias, infraestructura de AWS ni automatizaciones de despliegue fuera del alcance solicitado.

## Pruebas y forma de trabajar

- Para cada cambio, añade o actualiza pruebas relevantes. Prioriza pruebas unitarias para reglas de dominio y casos de uso; usa pruebas de integración cuando se necesite verificar SQL, transacciones o comportamiento real de una base de datos.
- Guarda los archivos de prueba `*_test.go` en la carpeta raíz `test/`, organizados por funcionalidad. Ejecuta las pruebas con `go test ./...`.
- No afirmes que una prueba pasó si no la ejecutaste. Cuando sea posible, ejecuta al menos `go test ./...` desde el entorno WSL del proyecto y comunica cualquier limitación.
- Mantén los cambios enfocados. No reviertas modificaciones existentes ni edites archivos ajenos a la tarea.
- Antes de una modificación amplia, resume brevemente el enfoque y sus supuestos. Si un requisito de negocio o una decisión de consistencia no está claro y afecta el diseño, pregunta antes de fijarlo.
- Al terminar, resume qué cambió, qué pruebas ejecutaste y qué quedó pendiente; señala riesgos o decisiones que requieran confirmación.
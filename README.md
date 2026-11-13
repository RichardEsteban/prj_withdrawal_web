# prj_withdrawal_web

Aplicación web en Go (v2.0.0) que refactoriza el proyecto `prj_withdrawal` (CLI v1.4.0):
**Retiro**, **Abono** e **Integridad de Transacción** con log en formato JSON.
Desarrollada con BDD/TDD de forma iterativa e incremental (DSL / Gherkin).

## Requisitos

- [Go 1.22 o superior](https://go.dev/dl). No usa dependencias externas.

## Ejecutar

```bash
go run ./webbank
```

Abrir <http://localhost:8080>. Opciones (para exponerlo en la red, usar `-addr :8080`):

| Bandera | Por defecto | Descripción |
|---|---|---|
| `-addr` | `127.0.0.1:8080` | Dirección de escucha (solo localhost; no abre puertos a la red ni muestra avisos del firewall) |
| `-log` | `data/transacciones.json` | Archivo JSON del historial |

Cuentas iniciales: Alice 234.56, Hamza 198.76, Carlos 500.00, Pedro 300.00.
Los saldos se restauran desde el log al reiniciar (el historial es la fuente de verdad).

## Pruebas

```bash
go vet ./...
go test ./... -count=1
```

Las pruebas replican los escenarios de `features/*.feature` (retiro, abono e integridad),
además de reversión por fallo del log, restauración de saldos, concurrencia y API HTTP.

## API JSON

| Método y ruta | Descripción |
|---|---|
| `GET /api/cuentas` | Lista de cuentas y saldos |
| `GET /api/transacciones` | Historial completo (log JSON) |
| `POST /api/transacciones` | Crea una transacción `{"titular":"Alice","tipo":"RETIRO","monto":200}` |

Respuestas de `POST`: `201` EXITOSA, `422` RECHAZADA o ERROR (queda registrada), `400` petición inválida,
`404` cuenta inexistente, `500` si no se pudo registrar (la transacción se revierte).

## Estructura

```
webbank/            punto de entrada (flags, composición)
internal/banco/     dominio: Cuenta, Transaccion, Registro (JSON atómico), Servicio (integridad)
internal/web/       capa HTTP: páginas HTML (plantilla incrustada) y API JSON
features/           especificación Gherkin (DSL)
```

## Reglas de negocio

- **RN-01** No se retira más que el saldo. **RN-04** Los montos deben ser mayores que cero.
- **RN-03** Atomicidad: si no se puede registrar, se revierte. **RN-07** Todo intento queda en el log
  (EXITOSA, RECHAZADA o ERROR). **RN-08** Solo las exitosas modifican el saldo. **RN-09** Id correlativo.

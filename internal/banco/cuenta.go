// Package banco contiene el dominio del proyecto: cuentas, transacciones,
// registro (log JSON) y el servicio que garantiza la integridad de cada operacion.
package banco

import "errors"

// Errores de dominio. Se usan con errors.Is para clasificar el resultado.
var (
	ErrMontoInvalido       = errors.New("el monto debe ser mayor que cero")
	ErrFondosInsuficientes = errors.New("fondos insuficientes")
	ErrCuentaNoExiste      = errors.New("la cuenta no existe")
	ErrTipoDesconocido     = errors.New("tipo de transaccion desconocido")
	ErrRegistro            = errors.New("no se pudo registrar la transaccion")
)

// Cuenta es la entidad de negocio (Retiro y Abono). No imprime ni persiste nada.
type Cuenta struct {
	Nombre string
	saldo  float64
}

// NuevaCuenta crea una cuenta con su saldo inicial.
func NuevaCuenta(nombre string, saldoInicial float64) *Cuenta {
	return &Cuenta{Nombre: nombre, saldo: saldoInicial}
}

// Saldo devuelve el saldo actual.
func (c *Cuenta) Saldo() float64 { return c.saldo }

// Retirar descuenta el monto. RN-01: no se puede retirar mas que el saldo.
// RN-04: el monto debe ser mayor que cero.
func (c *Cuenta) Retirar(monto float64) error {
	if monto <= 0 {
		return ErrMontoInvalido
	}
	if monto > c.saldo {
		return ErrFondosInsuficientes
	}
	c.saldo -= monto
	return nil
}

// Depositar abona el monto. RN-04: el monto debe ser mayor que cero.
func (c *Cuenta) Depositar(monto float64) error {
	if monto <= 0 {
		return ErrMontoInvalido
	}
	c.saldo += monto
	return nil
}

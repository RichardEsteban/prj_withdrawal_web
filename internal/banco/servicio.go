package banco

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Semilla define una cuenta inicial del sistema.
type Semilla struct {
	Nombre string
	Saldo  float64
}

// VistaCuenta es la proyeccion de una cuenta para la capa web.
type VistaCuenta struct {
	Nombre string  `json:"nombre"`
	Saldo  float64 `json:"saldo"`
}

// Servicio orquesta las operaciones y garantiza la integridad (US-04):
// RN-03 atomicidad, RN-07 todo intento queda registrado, RN-08 solo las
// exitosas modifican el saldo. Es seguro para uso concurrente.
type Servicio struct {
	mu      sync.Mutex
	cuentas map[string]*Cuenta
	orden   []string
	reg     Registro
	ahora   func() time.Time
}

// NuevoServicio crea el servicio y restaura los saldos a partir del log:
// el historial es la fuente de verdad, asi los saldos sobreviven a un reinicio.
func NuevoServicio(reg Registro, semillas []Semilla) (*Servicio, error) {
	s := &Servicio{cuentas: map[string]*Cuenta{}, reg: reg, ahora: time.Now}
	for _, sm := range semillas {
		s.cuentas[sm.Nombre] = NuevaCuenta(sm.Nombre, sm.Saldo)
		s.orden = append(s.orden, sm.Nombre)
	}
	historial, err := reg.Listar()
	if err != nil {
		return nil, fmt.Errorf("leyendo historial: %w", err)
	}
	for _, t := range historial {
		if c, ok := s.cuentas[t.Titular]; ok && t.Estado == EstadoExitosa {
			c.saldo = t.SaldoDespues
		}
	}
	return s, nil
}

// Procesar ejecuta un retiro o abono y deja siempre rastro en el log.
// Devuelve la transaccion registrada (EXITOSA, RECHAZADA o ERROR). Si el
// registro falla, revierte el saldo y devuelve ErrRegistro.
func (s *Servicio) Procesar(titular, tipo string, monto float64) (Transaccion, error) {
	if tipo != TipoRetiro && tipo != TipoAbono {
		return Transaccion{}, ErrTipoDesconocido
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cuentas[titular]
	if !ok {
		return Transaccion{}, ErrCuentaNoExiste
	}

	antes := c.saldo
	var err error
	if tipo == TipoRetiro {
		err = c.Retirar(monto)
	} else {
		err = c.Depositar(monto)
	}

	t := Transaccion{
		Fecha: s.ahora().Format(time.RFC3339), Titular: titular, Tipo: tipo,
		Monto: monto, SaldoAntes: antes, SaldoDespues: c.saldo,
	}
	switch {
	case err == nil:
		t.Estado, t.Mensaje = EstadoExitosa, "transaccion procesada"
	case errors.Is(err, ErrFondosInsuficientes):
		t.Estado, t.Mensaje = EstadoRechazada, err.Error()
	default:
		t.Estado, t.Mensaje = EstadoError, err.Error()
	}

	if gerr := s.reg.Guardar(&t); gerr != nil {
		c.saldo = antes // rollback: sin registro no hay transaccion
		t.SaldoDespues = antes
		return t, fmt.Errorf("%w: %v", ErrRegistro, gerr)
	}
	return t, nil
}

// Cuentas devuelve una copia de las cuentas en orden de creacion.
func (s *Servicio) Cuentas() []VistaCuenta {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]VistaCuenta, 0, len(s.orden))
	for _, n := range s.orden {
		out = append(out, VistaCuenta{Nombre: n, Saldo: s.cuentas[n].saldo})
	}
	return out
}

// Historial devuelve todas las transacciones registradas.
func (s *Servicio) Historial() ([]Transaccion, error) { return s.reg.Listar() }

package banco

import (
	"errors"
	"math"
	"path/filepath"
	"sync"
	"testing"
)

// registroMemoria es un doble de prueba del Registro.
type registroMemoria struct {
	mu    sync.Mutex
	datos []Transaccion
}

func (r *registroMemoria) Guardar(t *Transaccion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t.ID = len(r.datos) + 1
	r.datos = append(r.datos, *t)
	return nil
}
func (r *registroMemoria) Listar() ([]Transaccion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Transaccion(nil), r.datos...), nil
}

// registroRoto simula un log que no se puede escribir.
type registroRoto struct{}

func (registroRoto) Guardar(*Transaccion) error     { return errors.New("disco lleno") }
func (registroRoto) Listar() ([]Transaccion, error) { return nil, nil }

var semillas = []Semilla{{"Alice", 234.56}, {"Hamza", 198.76}}

func igual(a, b float64) bool { return math.Abs(a-b) < 0.001 }

func saldoDe(t *testing.T, s *Servicio, nombre string) float64 {
	t.Helper()
	for _, c := range s.Cuentas() {
		if c.Nombre == nombre {
			return c.Saldo
		}
	}
	t.Fatalf("cuenta %s no encontrada", nombre)
	return 0
}

// Escenarios Gherkin de retiro, abono e integridad (features/*.feature).
func TestEscenarios(t *testing.T) {
	casos := []struct {
		nombre, titular, tipo string
		monto                 float64
		estado                string
		saldoFinal            float64
	}{
		{"Esc 1: retiro exitoso dentro del saldo", "Alice", TipoRetiro, 200, EstadoExitosa, 34.56},
		{"Esc 2: retiro rechazado por exceder el saldo", "Hamza", TipoRetiro, 200, EstadoRechazada, 198.76},
		{"Esc 3/8: retiro con monto negativo es ERROR", "Alice", TipoRetiro, -20, EstadoError, 234.56},
		{"Esc 4: abono exitoso", "Alice", TipoAbono, 100, EstadoExitosa, 334.56},
		{"Esc 5: abono de monto invalido es ERROR", "Hamza", TipoAbono, -50, EstadoError, 198.76},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			reg := &registroMemoria{}
			s, err := NuevoServicio(reg, semillas)
			if err != nil {
				t.Fatal(err)
			}
			tx, err := s.Procesar(c.titular, c.tipo, c.monto)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if tx.Estado != c.estado {
				t.Errorf("estado = %s, esperado %s", tx.Estado, c.estado)
			}
			if got := saldoDe(t, s, c.titular); !igual(got, c.saldoFinal) {
				t.Errorf("saldo = %.2f, esperado %.2f", got, c.saldoFinal)
			}
			h, _ := reg.Listar()
			if len(h) != 1 || h[0].Estado != c.estado || h[0].ID != 1 {
				t.Errorf("el intento debe quedar en el log: %+v", h)
			}
		})
	}
}

// Esc 9: si el log falla, la transaccion se revierte (RN-03).
func TestReversionSiFallaElLog(t *testing.T) {
	s, _ := NuevoServicio(registroRoto{}, semillas)
	_, err := s.Procesar("Alice", TipoRetiro, 200)
	if !errors.Is(err, ErrRegistro) {
		t.Fatalf("error = %v, esperado ErrRegistro", err)
	}
	if got := saldoDe(t, s, "Alice"); !igual(got, 234.56) {
		t.Errorf("saldo = %.2f, debia revertirse a 234.56", got)
	}
}

func TestPeticionesInvalidasNoSeRegistran(t *testing.T) {
	reg := &registroMemoria{}
	s, _ := NuevoServicio(reg, semillas)
	if _, err := s.Procesar("Nadie", TipoRetiro, 10); !errors.Is(err, ErrCuentaNoExiste) {
		t.Errorf("cuenta inexistente: %v", err)
	}
	if _, err := s.Procesar("Alice", "TRANSFERENCIA", 10); !errors.Is(err, ErrTipoDesconocido) {
		t.Errorf("tipo desconocido: %v", err)
	}
	if h, _ := reg.Listar(); len(h) != 0 {
		t.Errorf("no debia registrarse nada, hay %d", len(h))
	}
}

// El historial es la fuente de verdad: los saldos se restauran tras reiniciar.
func TestRestauraSaldosDesdeElLog(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "data", "transacciones.json")
	s1, err := NuevoServicio(NuevoRegistroJSON(ruta), semillas)
	if err != nil {
		t.Fatal(err)
	}
	s1.Procesar("Alice", TipoRetiro, 200)
	s1.Procesar("Hamza", TipoRetiro, 999) // rechazada: no cambia el saldo

	s2, err := NuevoServicio(NuevoRegistroJSON(ruta), semillas)
	if err != nil {
		t.Fatal(err)
	}
	if got := saldoDe(t, s2, "Alice"); !igual(got, 34.56) {
		t.Errorf("Alice = %.2f, esperado 34.56", got)
	}
	if got := saldoDe(t, s2, "Hamza"); !igual(got, 198.76) {
		t.Errorf("Hamza = %.2f, esperado 198.76", got)
	}
	h, _ := s2.Historial()
	if len(h) != 2 || h[1].ID != 2 {
		t.Errorf("historial inesperado: %+v", h)
	}
}

// Seguridad ante concurrencia: 50 abonos simultaneos no pierden operaciones.
func TestConcurrencia(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "transacciones.json")
	s, _ := NuevoServicio(NuevoRegistroJSON(ruta), semillas)
	var wg sync.WaitGroup
	errs := make(chan error, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Procesar("Alice", TipoAbono, 1); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("error inesperado: %v", err)
	}
	if got := saldoDe(t, s, "Alice"); !igual(got, 284.56) {
		t.Errorf("saldo = %.2f, esperado 284.56", got)
	}
	h, _ := s.Historial()
	if len(h) != 50 {
		t.Errorf("entradas = %d, esperado 50", len(h))
	}
}

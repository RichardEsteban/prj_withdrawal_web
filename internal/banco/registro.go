package banco

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Tipos y estados de una transaccion.
const (
	TipoRetiro      = "RETIRO"
	TipoAbono       = "ABONO"
	EstadoExitosa   = "EXITOSA"
	EstadoRechazada = "RECHAZADA"
	EstadoError     = "ERROR"
)

// Transaccion es una entrada del log de auditoria (formato JSON).
type Transaccion struct {
	ID           int     `json:"id"`
	Fecha        string  `json:"fecha"`
	Titular      string  `json:"titular"`
	Tipo         string  `json:"tipo"`
	Monto        float64 `json:"monto"`
	SaldoAntes   float64 `json:"saldo_antes"`
	SaldoDespues float64 `json:"saldo_despues"`
	Estado       string  `json:"estado"`
	Mensaje      string  `json:"mensaje"`
}

// Registro abstrae el almacenamiento del log. Permite probar el servicio con
// implementaciones en memoria o que fallan a proposito.
type Registro interface {
	Guardar(t *Transaccion) error
	Listar() ([]Transaccion, error)
}

// RegistroJSON guarda el historial en un archivo JSON de forma atomica.
type RegistroJSON struct {
	mu   sync.Mutex
	ruta string
}

// NuevoRegistroJSON crea un registro asociado a un archivo.
func NuevoRegistroJSON(ruta string) *RegistroJSON { return &RegistroJSON{ruta: ruta} }

// Listar lee todo el historial (vacio si el archivo no existe).
func (r *RegistroJSON) Listar() ([]Transaccion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.leer()
}

func (r *RegistroJSON) leer() ([]Transaccion, error) {
	datos, err := os.ReadFile(r.ruta)
	if os.IsNotExist(err) {
		return []Transaccion{}, nil
	}
	if err != nil {
		return nil, err
	}
	var lista []Transaccion
	if err := json.Unmarshal(datos, &lista); err != nil {
		return nil, err
	}
	return lista, nil
}

// Guardar agrega la transaccion y asigna su id correlativo (RN-09).
// Escribe un archivo temporal completo y lo renombra: o queda todo o nada.
func (r *RegistroJSON) Guardar(t *Transaccion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	lista, err := r.leer()
	if err != nil {
		return err
	}
	t.ID = len(lista) + 1
	lista = append(lista, *t)
	datos, err := json.MarshalIndent(lista, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.ruta), 0o755); err != nil {
		return err
	}
	tmp := r.ruta + ".tmp"
	if err := os.WriteFile(tmp, datos, 0o644); err != nil {
		return err
	}
	return renombrar(tmp, r.ruta)
}

// renombrar reintenta unos instantes: en Windows el destino puede estar
// bloqueado momentaneamente por antivirus o indexadores.
func renombrar(origen, destino string) error {
	var err error
	for i := 0; i < 10; i++ {
		if err = os.Rename(origen, destino); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return err
}

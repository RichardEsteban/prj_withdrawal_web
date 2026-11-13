package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"prj_withdrawal_web/internal/banco"
)

func nuevoServidor(t *testing.T) http.Handler {
	t.Helper()
	reg := banco.NuevoRegistroJSON(filepath.Join(t.TempDir(), "transacciones.json"))
	svc, err := banco.NuevoServicio(reg, []banco.Semilla{{Nombre: "Alice", Saldo: 234.56}, {Nombre: "Hamza", Saldo: 198.76}})
	if err != nil {
		t.Fatal(err)
	}
	srv, err := Nuevo(svc)
	if err != nil {
		t.Fatal(err)
	}
	return srv.Rutas()
}

func peticion(h http.Handler, metodo, ruta, cuerpo, tipo string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	if tipo != "" {
		r.Header.Set("Content-Type", tipo)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestPaginaInicio(t *testing.T) {
	w := peticion(nuevoServidor(t), "GET", "/", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("codigo %d", w.Code)
	}
	for _, s := range []string{"Alice", "234.56", "Hamza", "Historial"} {
		if !strings.Contains(w.Body.String(), s) {
			t.Errorf("la pagina no contiene %q", s)
		}
	}
}

func TestAPITransacciones(t *testing.T) {
	h := nuevoServidor(t)
	casos := []struct {
		nombre, cuerpo string
		codigo         int
		estado         string
	}{
		{"retiro exitoso", `{"titular":"Alice","tipo":"RETIRO","monto":200}`, http.StatusCreated, "EXITOSA"},
		{"retiro rechazado", `{"titular":"Hamza","tipo":"RETIRO","monto":200}`, http.StatusUnprocessableEntity, "RECHAZADA"},
		{"abono invalido", `{"titular":"Hamza","tipo":"ABONO","monto":-50}`, http.StatusUnprocessableEntity, "ERROR"},
		{"abono exitoso", `{"titular":"Alice","tipo":"ABONO","monto":100}`, http.StatusCreated, "EXITOSA"},
		{"cuenta inexistente", `{"titular":"Nadie","tipo":"RETIRO","monto":1}`, http.StatusNotFound, ""},
		{"tipo desconocido", `{"titular":"Alice","tipo":"X","monto":1}`, http.StatusBadRequest, ""},
		{"json invalido", `{`, http.StatusBadRequest, ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			w := peticion(h, "POST", "/api/transacciones", c.cuerpo, "application/json")
			if w.Code != c.codigo {
				t.Fatalf("codigo %d, esperado %d (%s)", w.Code, c.codigo, w.Body.String())
			}
			if c.estado != "" {
				var tx banco.Transaccion
				json.Unmarshal(w.Body.Bytes(), &tx)
				if tx.Estado != c.estado {
					t.Errorf("estado %s, esperado %s", tx.Estado, c.estado)
				}
			}
		})
	}
	// Solo los 4 intentos validos quedan en el historial.
	w := peticion(h, "GET", "/api/transacciones", "", "")
	var lista []banco.Transaccion
	json.Unmarshal(w.Body.Bytes(), &lista)
	if len(lista) != 4 {
		t.Errorf("historial = %d entradas, esperado 4", len(lista))
	}
}

func TestFormularioRedirige(t *testing.T) {
	h := nuevoServidor(t)
	f := url.Values{"titular": {"Alice"}, "tipo": {"RETIRO"}, "monto": {"200"}}
	w := peticion(h, "POST", "/operar", f.Encode(), "application/x-www-form-urlencoded")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("codigo %d, esperado 303", w.Code)
	}
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "EXITOSA") {
		t.Errorf("redireccion inesperada: %s", loc)
	}
}

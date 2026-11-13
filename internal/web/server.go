// Package web expone el servicio bancario como aplicacion web (HTML) y API JSON.
package web

import (
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"prj_withdrawal_web/internal/banco"
)

//go:embed templates/index.html
var plantillas embed.FS

// Servidor agrupa las rutas HTTP.
type Servidor struct {
	svc *banco.Servicio
	tpl *template.Template
}

// Nuevo crea el servidor y compila la plantilla incrustada.
func Nuevo(svc *banco.Servicio) (*Servidor, error) {
	tpl, err := template.ParseFS(plantillas, "templates/index.html")
	if err != nil {
		return nil, err
	}
	return &Servidor{svc: svc, tpl: tpl}, nil
}

// Rutas devuelve el manejador con todas las rutas registradas.
func (s *Servidor) Rutas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.inicio)
	mux.HandleFunc("POST /operar", s.operar)
	mux.HandleFunc("GET /api/cuentas", s.apiCuentas)
	mux.HandleFunc("GET /api/transacciones", s.apiHistorial)
	mux.HandleFunc("POST /api/transacciones", s.apiProcesar)
	return mux
}

type vistaPagina struct {
	Cuentas   []banco.VistaCuenta
	Historial []banco.Transaccion
	Mensaje   string
	Estado    string
}

// inicio muestra cuentas, formulario e historial (mas reciente primero).
func (s *Servidor) inicio(w http.ResponseWriter, r *http.Request) {
	h, err := s.svc.Historial()
	if err != nil {
		http.Error(w, "no se pudo leer el historial", http.StatusInternalServerError)
		return
	}
	for i, j := 0, len(h)-1; i < j; i, j = i+1, j-1 {
		h[i], h[j] = h[j], h[i]
	}
	v := vistaPagina{Cuentas: s.svc.Cuentas(), Historial: h,
		Mensaje: r.URL.Query().Get("msg"), Estado: r.URL.Query().Get("estado")}
	if err := s.tpl.Execute(w, v); err != nil {
		log.Printf("plantilla: %v", err)
	}
}

// operar procesa el formulario HTML y redirige (patron Post/Redirect/Get).
func (s *Servidor) operar(w http.ResponseWriter, r *http.Request) {
	monto, err := strconv.ParseFloat(r.FormValue("monto"), 64)
	msg, estado := "", banco.EstadoError
	if err != nil {
		msg = "el monto no es un numero valido"
	} else {
		t, perr := s.svc.Procesar(r.FormValue("titular"), r.FormValue("tipo"), monto)
		if perr != nil {
			msg = perr.Error()
		} else {
			estado = t.Estado
			msg = "Transaccion " + strconv.Itoa(t.ID) + ": " + t.Tipo + " " + t.Estado + " (" + t.Mensaje + ")"
		}
	}
	http.Redirect(w, r, "/?estado="+url.QueryEscape(estado)+"&msg="+url.QueryEscape(msg), http.StatusSeeOther)
}

func escribirJSON(w http.ResponseWriter, codigo int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	json.NewEncoder(w).Encode(v)
}

func (s *Servidor) apiCuentas(w http.ResponseWriter, _ *http.Request) {
	escribirJSON(w, http.StatusOK, s.svc.Cuentas())
}

func (s *Servidor) apiHistorial(w http.ResponseWriter, _ *http.Request) {
	h, err := s.svc.Historial()
	if err != nil {
		escribirJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	escribirJSON(w, http.StatusOK, h)
}

// apiProcesar: 201 si EXITOSA, 422 si RECHAZADA o ERROR (queda en el log),
// 400/404 si la peticion es invalida, 500 si no se pudo registrar.
func (s *Servidor) apiProcesar(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Titular string  `json:"titular"`
		Tipo    string  `json:"tipo"`
		Monto   float64 `json:"monto"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		escribirJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON invalido"})
		return
	}
	t, err := s.svc.Procesar(req.Titular, req.Tipo, req.Monto)
	switch {
	case errors.Is(err, banco.ErrTipoDesconocido):
		escribirJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, banco.ErrCuentaNoExiste):
		escribirJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case err != nil:
		escribirJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	case t.Estado == banco.EstadoExitosa:
		escribirJSON(w, http.StatusCreated, t)
	default:
		escribirJSON(w, http.StatusUnprocessableEntity, t)
	}
}

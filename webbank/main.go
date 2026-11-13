// Comando webbank: servidor web del proyecto prj_withdrawal (v2.0.0, refactorizado).
package main

import (
	"flag"
	"log"
	"net/http"

	"prj_withdrawal_web/internal/banco"
	"prj_withdrawal_web/internal/web"
)

func main() {
	// Solo localhost por defecto: evita el aviso del Firewall de Windows y no expone la app en la red.
	addr := flag.String("addr", "127.0.0.1:8080", "direccion de escucha del servidor")
	logRuta := flag.String("log", "data/transacciones.json", "archivo JSON del log de transacciones")
	flag.Parse()

	svc, err := banco.NuevoServicio(banco.NuevoRegistroJSON(*logRuta), []banco.Semilla{
		{Nombre: "Alice", Saldo: 234.56},
		{Nombre: "Hamza", Saldo: 198.76},
		{Nombre: "Carlos", Saldo: 500.00},
		{Nombre: "Pedro", Saldo: 300.00},
	})
	if err != nil {
		log.Fatal(err)
	}
	srv, err := web.Nuevo(svc)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Banco web escuchando en %s (log: %s)", *addr, *logRuta)
	log.Fatal(http.ListenAndServe(*addr, srv.Rutas()))
}

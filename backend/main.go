package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"proyecto1/analyzer"
	"log"
	"net/http"
	"os"
)

// Estructuras para las peticiones/respuestas JSON
type ExecRequest struct {
	Commands string `json:"commands"`
}

type ExecResponse struct {
	Output string `json:"output"`
}

//go run main.go --server
func main() {
	// Verifica si hay un flag para iniciar en modo servidor
	if len(os.Args) > 1 && os.Args[1] == "--server" {
		// Inicia el servidor HTTP
		startServer()
		return
	}

	// Modo CLI interactivo normal
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		if line == "exit" {
			fmt.Println("Saliendo...")
			break
		}

		// Procesa la línea con nuestro analizador
		output := analyzer.ProcessCommands(line)
		fmt.Print(output)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error leyendo la entrada:", err)
	}
}

// Función que inicia el servidor HTTP
func startServer() {
	fmt.Println("Iniciando en modo servidor...")

	// Configura el manejador con CORS
	handler := http.NewServeMux()
	handler.HandleFunc("/execute", executeHandler)

	// Aplica middleware CORS
	corsHandler := enableCORS(handler)

	fmt.Println("Servidor ejecutando en http://localhost:3001")
	fmt.Println("Presiona Ctrl+C para detener")

	log.Fatal(http.ListenAndServe("0.0.0.0:3001", corsHandler))
}

// Middleware para habilitar CORS
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Configura los headers CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Para peticiones OPTIONS (preflight)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Manejador para el endpoint /execute
func executeHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        // Respuesta simple para pruebas remotas
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "message": "La API funciona! Usa POST con JSON para ejecutar comandos",
        })
        return
    case http.MethodPost:
        // Tu lógica original
        var req ExecRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
            return
        }

        log.Printf("Recibidos comandos: %s", req.Commands)

        output := analyzer.ProcessCommands(req.Commands)

        resp := ExecResponse{Output: output}
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    }
}
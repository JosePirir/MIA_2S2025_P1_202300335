package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExecuteListDisks lista los archivos .mia dentro del directorio indicado.
// Si path es vacío usa "./discos".
func ExecuteListDisks(path string) {
	if path == "" {
		path = "./discos"
	}
	dir := filepath.Clean(path)

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("Error al listar discos en '%s': %v\n", dir, err)
		return
	}

	found := false
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lname := strings.ToLower(name)
		if strings.HasSuffix(lname, ".mia") {
			fmt.Println(filepath.Join(dir, name))
			found = true
		}
	}

	if !found {
		fmt.Printf("No se encontraron discos (.mia) en: %s\n", dir)
	}
}

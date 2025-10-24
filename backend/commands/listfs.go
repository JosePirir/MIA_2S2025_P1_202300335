package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"

	"proyecto1/fs"
	"proyecto1/structs"
)

// ExecuteListFS lista las entradas dentro de una ruta en la partición indicada.
// Flags esperados (desde analyzer):
// -disk=<ruta del archivo .mia>
// -start=<offset en bytes donde comienza la partición>
// -path=<ruta interna dentro del FS> (ej. / o /carpeta)
func ExecuteListFS(diskPath string, startStr string, path string) {
	if diskPath == "" || startStr == "" {
		fmt.Println("Error: se requieren -disk y -start")
		return
	}
	start64, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		fmt.Println("Error: start no es un número válido:", err)
		return
	}

	f, err := os.Open(diskPath)
	if err != nil {
		fmt.Printf("Error al abrir disco '%s': %v\n", diskPath, err)
		return
	}
	defer f.Close()

	// Leer superbloque desde start
	var sb structs.Superblock
	if _, err := f.Seek(start64, 0); err != nil {
		fmt.Println("Error al posicionar superbloque:", err)
		return
	}
	if err := binary.Read(f, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer superbloque:", err)
		return
	}

	if path == "" {
		path = "/"
	}
	// Obtener inodo de la ruta
	inode, _, err := fs.FindInodeByPath(f, sb, path)
	if err != nil {
		fmt.Println("Error al resolver ruta:", err)
		return
	}

	// Debe ser carpeta
	if inode.I_type != 0 {
		fmt.Println("Error: la ruta no es un directorio")
		return
	}

	seen := map[string]bool{}
	// Recorrer bloques de carpeta
	for _, blockPtr := range inode.I_block {
		if blockPtr == -1 {
			continue
		}
		fb, err := fs.ReadFolderBlock(f, sb, blockPtr)
		if err != nil {
			fmt.Println("Error al leer bloque de carpeta:", err)
			return
		}
		for _, entry := range fb.B_content {
			if entry.B_inodo == -1 {
				continue
			}
			name := strings.TrimRight(string(entry.B_name[:]), "\x00")
			if name == "" {
				continue
			}
			// Evitar duplicados si varios bloques referencian lo mismo
			if seen[name] {
				continue
			}
			seen[name] = true
			// Leer inodo del entry para obtener tipo y tamaño
			childInode, err := fs.ReadInode(f, sb, entry.B_inodo)
			if err != nil {
				fmt.Printf("ERR|%s|0\n", name)
				continue
			}
			if childInode.I_type == 0 {
				// DIR|name
				fmt.Printf("DIR|%s\n", name)
			} else {
				// FILE|name|size
				fmt.Printf("FILE|%s|%d\n", name, childInode.I_size)
			}
		}
	}

	// Si no hubo entradas, no imprime nada (frontend mostrará vacío)
}

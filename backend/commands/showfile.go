package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"

	"proyecto1/fs"
	"proyecto1/structs"
)

// ExecuteShowFile imprime el contenido del archivo indicado en la partición
// Flags esperados:
// -disk=<ruta .mia>
// -start=<offset>
// -path=<ruta dentro del FS, p.ej. /foo/bar.txt>
func ExecuteShowFile(diskPath string, startStr string, path string) {
	if diskPath == "" || startStr == "" || path == "" {
		fmt.Println("Error: se requieren -disk, -start y -path")
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

	var sb structs.Superblock
	if _, err := f.Seek(start64, 0); err != nil {
		fmt.Println("Error al posicionar superbloque:", err)
		return
	}
	if err := binary.Read(f, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer superbloque:", err)
		return
	}

	inode, _, err := fs.FindInodeByPath(f, sb, path)
	if err != nil {
		fmt.Println("Error al resolver ruta:", err)
		return
	}
	if inode.I_type != 1 {
		fmt.Println("Error: la ruta no es un archivo")
		return
	}
	content, err := fs.ReadFileContent(f, sb, inode)
	if err != nil {
		fmt.Println("Error al leer contenido del archivo:", err)
		return
	}
	// Imprimir el contenido tal cual (frontend lo recibirá y podrá mostrarlo)
	fmt.Print(string(content))
}

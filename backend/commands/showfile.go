package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"proyecto1/fs"
	"proyecto1/structs"
)

// ExecuteShowFile imprime el contenido de un archivo dentro del sistema de archivos
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

	// Leer superbloque
	var sb structs.Superblock
	if _, err := f.Seek(start64, 0); err != nil {
		fmt.Println("Error al posicionar superbloque:", err)
		return
	}
	if err := binary.Read(f, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer superbloque:", err)
		return
	}

	// Buscar el inodo del archivo
	inode, _, err := fs.FindInodeByPath(f, sb, path)
	if err != nil {
		fmt.Println("Error al resolver ruta:", err)
		return
	}

	if inode.I_type != 1 {
		fmt.Println("Error: la ruta no es un archivo")
		return
	}

	// Leer los bloques de datos asociados al archivo
	var contentBuilder strings.Builder

	for _, blockNum := range inode.I_block {
		if blockNum == -1 {
			continue
		}

		blockPos := int64(sb.S_block_start) + int64(blockNum)*int64(sb.S_block_size)
		blockData := make([]byte, sb.S_block_size)

		if _, err := f.Seek(blockPos, 0); err != nil {
			fmt.Println("Error al posicionar en bloque:", err)
			return
		}

		if _, err := io.ReadFull(f, blockData); err != nil && err != io.EOF {
			fmt.Println("Error al leer bloque:", err)
			return
		}

		contentBuilder.Write(bytes.Trim(blockData, "\x00"))
	}

	// Mostrar el contenido completo del archivo
	fmt.Print(contentBuilder.String())
}

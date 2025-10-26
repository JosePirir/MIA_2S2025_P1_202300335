package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"proyecto1/fs"
	"proyecto1/structs"
)

// helper convierte bits de permisos (asumidos en los 9 bits bajos) a string rwxrwxrwx
func permsToString(p uint16) string {
	out := ""
	for g := 2; g >= 0; g-- {
		r := (p >> (g*3 + 2)) & 1
		w := (p >> (g*3 + 1)) & 1
		x := (p >> (g * 3)) & 1
		if r == 1 {
			out += "r"
		} else {
			out += "-"
		}
		if w == 1 {
			out += "w"
		} else {
			out += "-"
		}
		if x == 1 {
			out += "x"
		} else {
			out += "-"
		}
	}
	return out
}

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
	if sb.S_magic != 0xEF53 {
		fmt.Println("La partición no está formateada.")
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

	// recorrer bloques de carpeta
	seen := map[string]bool{}
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
			if name == "" || name == "." || name == ".." {
				continue
			}
			if seen[name] {
				continue
			}
			seen[name] = true

			childInode, err := fs.ReadInode(f, sb, entry.B_inodo)
			if err != nil {
				fmt.Printf("ERR|%s|0|-\n", name)
				continue
			}

			perms := permsToString(uint16(childInode.I_perm))
			if childInode.I_type == 0 {
				// DIR|name|0|perms
				fmt.Printf("DIR|%s|0|%s\n", filepath.Base(name), perms)
			} else {
				fmt.Printf("FILE|%s|%d|%s\n", filepath.Base(name), childInode.I_size, perms)
			}
		}
	}

	// Si no hubo entradas, no imprime nada (frontend mostrará vacío)
}

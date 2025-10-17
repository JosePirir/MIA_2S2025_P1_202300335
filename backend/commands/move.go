package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
)

func ExecuteMove(path string, destino string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: debes iniciar sesión para usar move.")
		return
	}

	var mountedPartition *state.MountedPartition
	for _, mp := range state.GlobalMountedPartitions {
		if mp.ID == state.CurrentSession.PartitionID {
			mountedPartition = &mp
			break
		}
	}
	if mountedPartition == nil {
		fmt.Println("Error: no se encontró la partición activa.")
		return
	}

	file, err := os.OpenFile(mountedPartition.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	// Leer superbloque
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// Buscar inodo origen
	inodeOrigen, parentOrigen, errOrigen := fs.FindInodeByPath(file, sb, path)
	if errOrigen != nil {
		fmt.Println("Error: la ruta origen no existe.")
		return
	}

	// Verificar permisos de escritura en el origen
	uid, gid, _ := getUserIDs(file, sb, state.CurrentSession.User)
	if !tienePermisoEscritura(inodeOrigen, uid, gid) {
		fmt.Println("Error: no tienes permiso de escritura sobre el archivo o carpeta origen.")
		return
	}

	// Buscar destino
	inodeDestino, _, errDestino := fs.FindInodeByPath(file, sb, destino)
	if errDestino != nil {
		fmt.Println("Error: la ruta destino no existe.")
		return
	}

	// Verificar que el destino sea una carpeta
	if inodeDestino.I_type != 0 {
		fmt.Println("Error: el destino no es una carpeta.")
		return
	}

	// Verificar permisos de escritura sobre la carpeta destino
	if !tienePermisoEscritura(inodeDestino, uid, gid) {
		fmt.Println("Error: no tienes permiso de escritura sobre la carpeta destino.")
		return
	}

	// Obtener nombre del archivo/carpeta a mover
	nombreOrigen := getLastNameFromPath(path)

	// --- 1. Eliminar referencia del padre original ---
	parentInode, _ := fs.ReadInode(file, sb, parentOrigen)
	for _, blockNum := range parentInode.I_block {
		if blockNum == -1 {
			continue
		}
		fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
		for i := range fb.B_content {
			entryName := string(bytes.Trim(fb.B_content[i].B_name[:], "\x00"))
			if entryName == nombreOrigen {
				fb.B_content[i].B_inodo = -1
				copy(fb.B_content[i].B_name[:], make([]byte, len(fb.B_content[i].B_name)))
				fs.WriteFolderBlock(file, sb, blockNum, fb)
				break
			}
		}
	}

	// --- 2. Agregar referencia en el nuevo padre ---
	inserted := false
	for _, blockNum := range inodeDestino.I_block {
		if blockNum == -1 {
			continue
		}
		destFB, _ := fs.ReadFolderBlock(file, sb, blockNum)
		for i := range destFB.B_content {
			if destFB.B_content[i].B_inodo == -1 {
				copy(destFB.B_content[i].B_name[:], []byte(nombreOrigen))
				destFB.B_content[i].B_inodo = parentOrigen // Se mantiene el mismo índice del inodo movido
				fs.WriteFolderBlock(file, sb, blockNum, destFB)
				inserted = true
				break
			}
		}
		if inserted {
			break
		}
	}

	if !inserted {
		fmt.Println("Error: no hay espacio disponible en la carpeta destino.")
		return
	}

	fmt.Printf("Elemento '%s' movido correctamente a '%s'.\n", path, destino)
}

// --- Función auxiliar ---
func getLastNameFromPath(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	return parts[len(parts)-1]
}

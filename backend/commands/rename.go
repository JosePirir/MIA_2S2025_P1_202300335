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

func ExecuteRename(path string, newName string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: debes iniciar sesión para usar rename.")
		return
	}

	// --- 1. Obtener la partición activa ---
	var mountedPartition *state.MountedPartition
	for _, p := range state.GlobalMountedPartitions {
		if p.ID == state.CurrentSession.PartitionID {
			mountedPartition = &p
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

	// --- 2. Leer superbloque ---
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	uid, gid, _ := getUserIDs(file, sb, state.CurrentSession.User)

	// --- 3. Separar ruta ---
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) == 0 {
		fmt.Println("Error: ruta inválida.")
		return
	}
	targetName := pathParts[len(pathParts)-1] // nombre actual
	parentPath := strings.Join(pathParts[:len(pathParts)-1], "/")

	// --- 4. Buscar carpeta padre ---
	currentInodeIndex := int32(0)
	if parentPath != "" {
		// Navegar hasta el inodo padre
		parts := strings.Split(parentPath, "/")
		for _, p := range parts {
			inode, _ := fs.ReadInode(file, sb, currentInodeIndex)
			found := false
			for _, blockNum := range inode.I_block {
				if blockNum == -1 {
					continue
				}
				fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
				for _, entry := range fb.B_content {
					name := string(bytes.Trim(entry.B_name[:], "\x00"))
					if name == p && entry.B_inodo != -1 {
						currentInodeIndex = entry.B_inodo
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				fmt.Println("Error: no se encontró la carpeta padre.")
				return
			}
		}
	}

	parentInode, _ := fs.ReadInode(file, sb, currentInodeIndex)
	if !tienePermisoEscritura(parentInode, uid, gid) {
		fmt.Println("Error: no tienes permiso de escritura en la carpeta padre.")
		return
	}

	// --- 5. Verificar existencia del nuevo nombre ---
	for _, blockNum := range parentInode.I_block {
		if blockNum == -1 {
			continue
		}
		fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
		for _, entry := range fb.B_content {
			name := string(bytes.Trim(entry.B_name[:], "\x00"))
			if name == newName {
				fmt.Println("Error: ya existe un archivo o carpeta con ese nombre en esta ubicación.")
				return
			}
		}
	}

	// --- 6. Buscar el archivo/carpeta a renombrar ---
	foundEntry := false
	for _, blockNum := range parentInode.I_block {
		if blockNum == -1 {
			continue
		}
		fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
		for i, entry := range fb.B_content {
			name := string(bytes.Trim(entry.B_name[:], "\x00"))
			if name == targetName {
				copy(fb.B_content[i].B_name[:], []byte(newName))
				fs.WriteFolderBlock(file, sb, blockNum, fb)
				foundEntry = true
				break
			}
		}
		if foundEntry {
			break
		}
	}

	if !foundEntry {
		fmt.Println("Error: no se encontró el archivo o carpeta especificado.")
		return
	}

	fmt.Printf("Nombre cambiado correctamente: '%s' → '%s'\n", targetName, newName)
}

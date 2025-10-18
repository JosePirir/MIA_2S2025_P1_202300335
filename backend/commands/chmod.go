package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"

	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
)

// ======================== ExecuteChmod ========================
//
// Cambia los permisos (ugo) de un archivo o carpeta.
// Solo el usuario root puede ejecutarlo.
// Si se usa -r y el path apunta a una carpeta, el cambio será recursivo.
//
func ExecuteChmod(path string, ugo string, recursive bool) {
	// Validar sesión activa
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar chmod.")
		return
	}

	// Solo root puede ejecutar chmod
	if state.CurrentSession.User != "root" {
		fmt.Println("Error: Solo el usuario root puede ejecutar chmod.")
		return
	}

	// Validar parámetro -ugo
	if len(ugo) != 3 {
		fmt.Println("Error: El parámetro -ugo debe tener exactamente 3 dígitos (U,G,O).")
		return
	}
	for _, c := range ugo {
		if c < '0' || c > '7' {
			fmt.Println("Error: Cada dígito del parámetro -ugo debe estar entre 0 y 7.")
			return
		}
	}

	permInt, err := strconv.Atoi(ugo)
	if err != nil {
		fmt.Println("Error: No se pudo convertir el parámetro -ugo a número entero.")
		return
	}

	// Buscar la partición montada actual
	var mountedPartition *state.MountedPartition
	for _, mp := range state.GlobalMountedPartitions {
		if mp.ID == state.CurrentSession.PartitionID {
			mountedPartition = &mp
			break
		}
	}
	if mountedPartition == nil {
		fmt.Println("Error: No se encontró la partición activa.")
		return
	}

	// Abrir el archivo del disco
	file, err := os.OpenFile(mountedPartition.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	// Leer el superbloque
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// Buscar el inodo por ruta
	inode, inodeIndex, err := fs.FindInodeByPath(file, sb, path)
	if err != nil {
		fmt.Println("Error al buscar la ruta:", err)
		return
	}

	// Cambiar permisos del inodo principal
	inode.I_perm = int32(permInt)
	fs.WriteInode(file, sb, inodeIndex, inode)

	fmt.Println("Permisos cambiados a", ugo, "en", path)

	// Si se especificó -r y el inodo es carpeta, aplicar recursivamente
	if recursive && inode.I_type == 0 {
		aplicarChmodRecursivo(file, sb, inodeIndex, int32(permInt))
	}
}

// ======================== aplicarChmodRecursivo ========================
//
// Recorre todos los subdirectorios y archivos dentro de un inodo tipo carpeta,
// cambiando sus permisos.
//
func aplicarChmodRecursivo(file *os.File, sb structs.Superblock, inodeIndex int32, perm int32) {
	inode, err := fs.ReadInode(file, sb, inodeIndex)
	if err != nil {
		return
	}

	for _, block := range inode.I_block {
		if block == -1 {
			continue
		}

		fb, err := fs.ReadFolderBlock(file, sb, block)
		if err != nil {
			continue
		}

		for _, entry := range fb.B_content {
			if entry.B_inodo == -1 {
				continue
			}

			name := strings.Trim(string(entry.B_name[:]), "\x00 ")
			if name == "." || name == ".." {
				continue
			}

			childInode, err := fs.ReadInode(file, sb, entry.B_inodo)
			if err != nil {
				continue
			}

			childInode.I_perm = perm
			fs.WriteInode(file, sb, entry.B_inodo, childInode)

			// Si es carpeta, aplicar recursivamente
			if childInode.I_type == 0 {
				aplicarChmodRecursivo(file, sb, entry.B_inodo, perm)
			}
		}
	}
}
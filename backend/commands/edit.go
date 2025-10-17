package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"
	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
)

// ExecuteEdit permite editar el contenido de un archivo existente.
func ExecuteEdit(path string, cont string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar edit.")
		return
	}

	// Buscar partición activa
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

	file, err := os.OpenFile(mountedPartition.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	// Leer superbloque
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	binary.Read(file, binary.BigEndian, &sb)

	uid, gid, _ := getUserIDs(file, sb, state.CurrentSession.User)

	// Buscar el inodo del archivo
	inode, _, err := fs.FindInodeByPath(file, sb, path)
	if err != nil {
		fmt.Println("Error: el archivo no existe:", path)
		return
	}

	// Validar que sea un archivo (no carpeta)
	if inode.I_type == 0 {
		fmt.Println("Error: no puedes editar una carpeta.")
		return
	}

	// Validar permisos de lectura y escritura
	if !tienePermisoEscritura(inode, uid, gid) {
		fmt.Println("Error: no tienes permiso para editar este archivo.")
		return
	}

	// Leer contenido nuevo desde archivo real del sistema operativo
	fileContent, err := os.ReadFile(cont)
	if err != nil {
		fmt.Println("Error: no se pudo leer el archivo de contenido:", err)
		return
	}

	newContent := fileContent
	inode.I_size = int32(len(newContent))
	inode.I_mtime = time.Now().Unix()

	// Limpiar bloques viejos
	for _, blockNum := range inode.I_block {
		if blockNum != -1 {
			fs.MarkBlockAsFree(file, sb, blockNum, mountedPartition.Start)
		}
	}

	// Escribir nuevo contenido
	blockSize := len(structs.FileBlock{}.B_content)
	offset := 0
	for i := 0; offset < len(newContent) && i < len(inode.I_block); i++ {
		blockIndex, _ := fs.FindFreeBlock(file, sb)
		fs.MarkBlockAsUsed(file, sb, blockIndex)

		end := offset + blockSize
		if end > len(newContent) {
			end = len(newContent)
		}

		var fb structs.FileBlock
		copy(fb.B_content[:], newContent[offset:end])
		offset = end

		inode.I_block[i] = blockIndex
		fs.WriteFileBlock(file, sb, blockIndex, fb)
	}

	// Guardar inodo actualizado
	fs.WriteInode(file, sb, inode.I_uid, inode)

	fmt.Println("Archivo editado correctamente:", path)
}

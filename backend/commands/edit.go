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

func ExecuteEdit(path string, cont string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar edit.")
		return
	}

	// Partición activa
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
	binary.Read(file, binary.BigEndian, &sb)

	uid, gid, _ := getUserIDs(file, sb, state.CurrentSession.User)

	// Buscar inodo
	inode, inodeIndex, err := fs.FindInodeByPath(file, sb, path)
	if err != nil {
		fmt.Println("Error: el archivo no existe:", path)
		return
	}

	if inode.I_type == 0 {
		fmt.Println("Error: no puedes editar una carpeta.")
		return
	}

	// Permisos lectura y escritura
	if !tienePermisoLectura(inode, uid, gid) || !tienePermisoEscritura(inode, uid, gid) {
		fmt.Println("Error: no tienes permisos de lectura/escritura sobre este archivo.")
		return
	}

	// Leer contenido del archivo externo
	newContent, err := os.ReadFile(cont)
	if err != nil {
		fmt.Println("Error: no se pudo leer el archivo de contenido:", err)
		return
	}

	// Liberar bloques antiguos
	for i := range inode.I_block {
		if inode.I_block[i] != -1 {
			fs.MarkBlockAsFree(file, sb, inode.I_block[i], mountedPartition.Start)
			inode.I_block[i] = -1
		}
	}

	// Escribir nuevo contenido
	blockSize := len(structs.FileBlock{}.B_content)
	offset := 0
	for offset < len(newContent) {
		blockIndex, _ := fs.FindFreeBlock(file, sb)
		fs.MarkBlockAsUsed(file, sb, blockIndex)

		var fb structs.FileBlock
		end := offset + blockSize
		if end > len(newContent) {
			end = len(newContent)
		}
		copy(fb.B_content[:], newContent[offset:end])
		fs.WriteFileBlock(file, sb, blockIndex, fb)

		// Asignar al primer bloque libre en inode
		assigned := false
		for i := range inode.I_block {
			if inode.I_block[i] == -1 {
				inode.I_block[i] = blockIndex
				assigned = true
				break
			}
		}
		if !assigned {
			fmt.Println("Error: no hay suficiente espacio en los punteros del inodo.")
			break
		}
		offset = end
	}

	// Actualizar inodo
	inode.I_size = int32(len(newContent))
	inode.I_mtime = time.Now().Unix()
	fs.WriteInode(file, sb, inodeIndex, inode)

	fmt.Println("Archivo editado correctamente:", path)
}

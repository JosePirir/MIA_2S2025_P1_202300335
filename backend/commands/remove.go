package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
	"time"
	
)

// ExecuteRemove elimina un archivo o carpeta si el usuario tiene permisos.
func ExecuteRemove(filePath string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar rm.")
		return
	}

	// --- Obtener partición activa ---
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

	// --- Leer superbloque ---
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	uid, gid, _ := getUserIDs(file, sb, state.CurrentSession.User)

	// --- Buscar el inodo del archivo/carpeta ---
	inode, _, err := fs.FindInodeByPath(file, sb, filePath)
	if err != nil {
		fmt.Println("Error: el archivo o carpeta no existe:", filePath)
		return
	}

	// --- Validar permisos ---
	if !tienePermisoEscritura(inode, uid, gid) {
		fmt.Println("Error: no tienes permisos para eliminar este archivo o carpeta.")
		return
	}

	// --- Obtener inodo padre ---
	parentPath := path.Dir(filePath)
	parentInode, parentIndex, err := fs.FindInodeByPath(file, sb, parentPath)
	if err != nil {
		fmt.Println("Error: no se encontró la carpeta padre:", parentPath)
		return
	}

	// --- Eliminar archivo o carpeta ---
	if inode.I_type == 0 {
		// Carpeta → eliminación recursiva
		removeFolderRecursively(file, sb, inode.I_uid, mountedPartition.Start)
	} else {
		// Archivo
		removeFile(file, sb, inode.I_uid, mountedPartition.Start)
	}

	// --- Eliminar entrada del padre ---
	removeEntryFromParent(file, sb, parentIndex, path.Base(filePath))

	// --- Actualizar timestamp del padre ---
	parentInode.I_mtime = time.Now().Unix()
	fs.WriteInode(file, sb, parentIndex, parentInode)

	addJournalEntry(file, sb, mountedPartition.Start, "REMOVE", filePath, "-")

	fmt.Println("Eliminación completada exitosamente:", filePath)
}

// removeFile elimina los bloques y el inodo de un archivo.
func removeFile(file *os.File, sb structs.Superblock, inodeIndex int32, sbStart int64) {
	inode, _ := fs.ReadInode(file, sb, inodeIndex)
	for _, blockNum := range inode.I_block {
		if blockNum != -1 {
			fs.MarkBlockAsFree(file, sb, blockNum, sbStart)
		}
	}
	fs.MarkInodeAsFree(file, sb, inodeIndex, sbStart)
}

// canDeleteFolderRecursively verifica que el usuario tenga permiso de escritura en todos los elementos.
func canDeleteFolderRecursively(file *os.File, sb structs.Superblock, inode structs.Inode, uid, gid int32) bool {
	for _, blockNum := range inode.I_block {
		if blockNum == -1 {
			continue
		}
		fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
		for _, entry := range fb.B_content {
			if entry.B_inodo == -1 {
				continue
			}
			childInode, _ := fs.ReadInode(file, sb, entry.B_inodo)
			name := string(bytes.Trim(entry.B_name[:], "\x00"))
			if name == "." || name == ".." {
				continue
			}
			if !tienePermisoEscritura(childInode, uid, gid) {
				return false
			}
			if childInode.I_type == 0 {
				if !canDeleteFolderRecursively(file, sb, childInode, uid, gid) {
					return false
				}
			}
		}
	}
	return true
}

// removeFolderRecursively elimina todos los archivos e inodos de una carpeta.
func removeFolderRecursively(file *os.File, sb structs.Superblock, inodeIndex int32, sbStart int64) {
	inode, _ := fs.ReadInode(file, sb, inodeIndex)

	for _, blockNum := range inode.I_block {
		if blockNum == -1 {
			continue
		}
		fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
		for _, entry := range fb.B_content {
			if entry.B_inodo == -1 {
				continue
			}
			name := string(bytes.Trim(entry.B_name[:], "\x00"))
			if name == "." || name == ".." {
				continue
			}
			childInode, _ := fs.ReadInode(file, sb, entry.B_inodo)
			if childInode.I_type == 0 {
				removeFolderRecursively(file, sb, entry.B_inodo, sbStart)
			} else {
				removeFile(file, sb, entry.B_inodo, sbStart)
			}
		}
		fs.MarkBlockAsFree(file, sb, blockNum, sbStart)
	}
	fs.MarkInodeAsFree(file, sb, inodeIndex, sbStart)
}

// removeEntryFromParent borra la entrada de un archivo/carpeta del bloque de su carpeta padre.
func removeEntryFromParent(file *os.File, sb structs.Superblock, parentIndex int32, name string) {
	parent, _ := fs.ReadInode(file, sb, parentIndex)
	for _, blockNum := range parent.I_block {
		if blockNum == -1 {
			continue
		}
		fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
		for idx, entry := range fb.B_content {
			if string(bytes.Trim(entry.B_name[:], "\x00")) == name {
				fb.B_content[idx].B_inodo = -1
				fs.WriteFolderBlock(file, sb, blockNum, fb)
				return
			}
		}
	}
}

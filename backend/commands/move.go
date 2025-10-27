package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"time"

	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
)

// ExecuteMove: mueve un archivo o carpeta a otro destino dentro de la misma partición.
func ExecuteMove(srcPath string, destPath string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: debes iniciar sesión para usar move.")
		return
	}

	// --- Obtener partición activa ---
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

	// --- Leer superbloque ---
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	uid, gid, _ := getUserIDs(file, sb, state.CurrentSession.User)

	// --- Buscar origen ---
	srcInode, srcIndex, err := fs.FindInodeByPath(file, sb, srcPath)
	if err != nil {
		fmt.Println("Error: no se encontró la ruta origen:", srcPath)
		return
	}

	if !tienePermisoEscritura(srcInode, uid, gid) {
		fmt.Println("Error: no tienes permiso de escritura sobre el origen.")
		return
	}

	// --- Obtener padre del origen ---
	srcParentPath := path.Dir(srcPath)
	srcParentInode, srcParentIndex, err := fs.FindInodeByPath(file, sb, srcParentPath)
	if err != nil {
		fmt.Println("Error: no se encontró la carpeta padre del origen:", srcParentPath)
		return
	}

	// --- Determinar destino ---
	var destParentInode structs.Inode
	var destParentIndex int32
	var destName string

	destInode, destIdx, errDest := fs.FindInodeByPath(file, sb, destPath)
	if errDest == nil {
		if destInode.I_type == 0 {
			// destPath es carpeta: mover dentro
			destParentInode = destInode
			destParentIndex = destIdx
			destName = path.Base(srcPath)
		} else {
			// destPath es archivo: sobreescribir
			parentPath := path.Dir(destPath)
			pInode, pIdx, err := fs.FindInodeByPath(file, sb, parentPath)
			if err != nil {
				fmt.Println("Error: no se encontró la carpeta padre del destino:", parentPath)
				return
			}
			if pInode.I_type != 0 {
				fmt.Println("Error: el padre del destino no es una carpeta:", parentPath)
				return
			}
			destParentInode = pInode
			destParentIndex = pIdx
			destName = path.Base(destPath)
		}
	} else {
		// destPath no existe: buscar padre
		parentPath := path.Dir(destPath)
		pInode, pIdx, err := fs.FindInodeByPath(file, sb, parentPath)
		if err != nil {
			fmt.Println("Error: la carpeta destino no existe:", parentPath)
			return
		}
		if pInode.I_type != 0 {
			fmt.Println("Error: el destino debe ser una carpeta.")
			return
		}
		destParentInode = pInode
		destParentIndex = pIdx
		destName = path.Base(destPath)
	}

	// --- Verificar permisos escritura en destino ---
	if !tienePermisoEscritura(destParentInode, uid, gid) {
		fmt.Println("Error: no tienes permiso de escritura en la carpeta destino.")
		return
	}

	// --- Agregar entrada al destino sin duplicar datos ---
	if err := addEntryToParent(file, sb, destParentIndex, destName, srcIndex); err != nil {
		fmt.Println("Error agregando entrada al destino:", err)
		return
	}

	// --- Eliminar entrada del padre original ---
	removeEntryFromParent(file, sb, srcParentIndex, path.Base(srcPath))

	// --- Actualizar timestamps ---
	now := time.Now().Unix()
	srcInode.I_mtime = now
	fs.WriteInode(file, sb, srcIndex, srcInode)

	srcParentInode.I_mtime = now
	fs.WriteInode(file, sb, srcParentIndex, srcParentInode)

	destParentInode.I_mtime = now
	fs.WriteInode(file, sb, destParentIndex, destParentInode)

	fmt.Println("Movimiento completado correctamente.")
	addJournalEntry(file, sb, mountedPartition.Start, "MOVE", srcPath+" -> "+destPath, "-")
}

// addEntryToParent agrega un inodo existente a una carpeta (sin duplicar).
func addEntryToParent(file *os.File, sb structs.Superblock, parentIndex int32, name string, inodeIndex int32) error {
	parentInode, err := fs.ReadInode(file, sb, parentIndex)
	if err != nil {
		return err
	}

	inserted := false
	for _, blockNum := range parentInode.I_block {
		if blockNum == -1 {
			continue
		}
		fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
		for i, entry := range fb.B_content {
			if entry.B_inodo == -1 {
				copy(fb.B_content[i].B_name[:], []byte(name))
				fb.B_content[i].B_inodo = inodeIndex
				fs.WriteFolderBlock(file, sb, blockNum, fb)
				inserted = true
				break
			}
		}
		if inserted {
			break
		}
	}

	if !inserted {
		// Crear nuevo bloque de carpeta si no hay espacio
		newBlockIndex, _ := fs.FindFreeBlock(file, sb)
		fs.MarkBlockAsUsed(file, sb, newBlockIndex)

		var fb structs.FolderBlock
		for i := range fb.B_content {
			fb.B_content[i].B_inodo = -1
		}
		copy(fb.B_content[0].B_name[:], []byte(name))
		fb.B_content[0].B_inodo = inodeIndex
		fs.WriteFolderBlock(file, sb, newBlockIndex, fb)

		wrote := false
		for i := range parentInode.I_block {
			if parentInode.I_block[i] == -1 {
				parentInode.I_block[i] = newBlockIndex
				fs.WriteInode(file, sb, parentIndex, parentInode)
				wrote = true
				break
			}
		}
		if !wrote {
			return fmt.Errorf("no hay espacio en el inodo padre para agregar bloque")
		}
	}

	return nil
}

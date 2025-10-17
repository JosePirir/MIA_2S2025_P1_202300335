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
)

func ExecuteCopy(srcPath string, destPath string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: debes iniciar sesión para usar copy.")
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

	// --- 3. Buscar origen ---
	srcInode, _, err := fs.FindInodeByPath(file, sb, srcPath)
	if err != nil {
		fmt.Println("Error: no se encontró la ruta origen:", srcPath)
		return
	}

	// --- 4. Verificar permisos de lectura ---
	if !tienePermisoLectura(srcInode, uid, gid) {
		fmt.Println("Error: no tienes permiso de lectura sobre el archivo o carpeta de origen.")
		return
	}

	// --- 5. Buscar destino ---
	destInode, _, err := fs.FindInodeByPath(file, sb, destPath)
	if err != nil {
		fmt.Println("Error: la carpeta de destino no existe:", destPath)
		return
	}

	// --- 6. Verificar que destino sea carpeta ---
	if destInode.I_type != 0 {
		fmt.Println("Error: el destino debe ser una carpeta.")
		return
	}

	if !tienePermisoEscritura(destInode, uid, gid) {
		fmt.Println("Error: no tienes permiso de escritura en la carpeta destino.")
		return
	}

	// --- 7. Copiar recursivamente ---
	copyRecursive(file, sb, srcInode, path.Base(srcPath), destInode, uid, gid)

	fmt.Println("Copia completada correctamente.")
}

// ------------------------------------------------------------
// Función auxiliar recursiva: copia archivos y carpetas
// ------------------------------------------------------------
func copyRecursive(file *os.File, sb structs.Superblock, srcInode structs.Inode, name string, destInode structs.Inode, uid int32, gid int32) {
	if srcInode.I_type == 1 {
		// Es archivo
		copyFile(file, sb, srcInode, name, destInode, uid, gid)
	} else {
		// Es carpeta
		copyFolder(file, sb, srcInode, name, destInode, uid, gid)
	}
}

// ------------------------------------------------------------
// Copiar archivo individual
// ------------------------------------------------------------
func copyFile(file *os.File, sb structs.Superblock, srcInode structs.Inode, name string, destInode structs.Inode, uid int32, gid int32) {
	if !tienePermisoLectura(srcInode, uid, gid) {
		fmt.Println("Saltando archivo (sin permisos de lectura):", name)
		return
	}

	// Leer contenido
	var content bytes.Buffer
	for _, blockNum := range srcInode.I_block {
		if blockNum == -1 {
			continue
		}
		fb, _ := fs.ReadFileBlock(file, sb, blockNum)
		content.Write(bytes.Trim(fb.B_content[:], "\x00"))
	}

	// Crear nuevo archivo con el mismo contenido
	tmpPath := "/tmp_copy_" + name // ruta temporal
	os.WriteFile(tmpPath, content.Bytes(), 0644)

	ExecuteMkfile(path.Join("/", name), false, 0, tmpPath)
	os.Remove(tmpPath)
}

// ------------------------------------------------------------
// Copiar carpeta (recursivo)
// ------------------------------------------------------------
func copyFolder(file *os.File, sb structs.Superblock, srcInode structs.Inode, folderName string, destInode structs.Inode, uid int32, gid int32) {
	if !tienePermisoLectura(srcInode, uid, gid) {
		fmt.Println("Saltando carpeta (sin permisos de lectura):", folderName)
		return
	}

	// Crear la carpeta destino
	destFolderPath := path.Join("/", folderName)
	ExecuteMkdir(destFolderPath, false)

	// Leer todos los contenidos del directorio fuente
	for _, blockNum := range srcInode.I_block {
		if blockNum == -1 {
			continue
		}
		fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
		for _, entry := range fb.B_content {
			entryName := string(bytes.Trim(entry.B_name[:], "\x00"))
			if entryName == "" || entry.B_inodo == -1 || entryName == "." || entryName == ".." {
				continue
			}

			entryInode, _ := fs.ReadInode(file, sb, entry.B_inodo)
			copyRecursive(file, sb, entryInode, entryName, destInode, uid, gid)
		}
	}
}


// --------------- PERMISOS--------------

/*func tienePermisoLectura(inode structs.Inode, uid int32, gid int32) bool {
	permStr := strconv.Itoa(int(inode.I_perm))
	if len(permStr) < 3 {
		permStr = "0" + permStr
	}
	userPerm, _ := strconv.Atoi(string(permStr[0]))
	groupPerm, _ := strconv.Atoi(string(permStr[1]))
	otherPerm, _ := strconv.Atoi(string(permStr[2]))

	if inode.I_uid == uid {
		return (userPerm & 4) != 0 // bit 4 = lectura
	}
	if inode.I_gid == gid {
		return (groupPerm & 4) != 0
	}
	return (otherPerm & 4) != 0
}*/

// --- Verifica permiso de escritura ---
/*func tienePermisoEscritura(inode structs.Inode, uid int32, gid int32) bool {
	permStr := strconv.Itoa(int(inode.I_perm))
	if len(permStr) < 3 {
		permStr = "0" + permStr
	}
	userPerm, _ := strconv.Atoi(string(permStr[0]))
	groupPerm, _ := strconv.Atoi(string(permStr[1]))
	otherPerm, _ := strconv.Atoi(string(permStr[2]))

	if inode.I_uid == uid {
		return (userPerm & 2) != 0 // bit 2 = escritura
	}
	if inode.I_gid == gid {
		return (groupPerm & 2) != 0
	}
	return (otherPerm & 2) != 0
}*/

// --- Verifica permiso de ejecución ---
/*func tienePermisoEjecucion(inode structs.Inode, uid int32, gid int32) bool {
	permStr := strconv.Itoa(int(inode.I_perm))
	if len(permStr) < 3 {
		permStr = "0" + permStr
	}
	userPerm, _ := strconv.Atoi(string(permStr[0]))
	groupPerm, _ := strconv.Atoi(string(permStr[1]))
	otherPerm, _ := strconv.Atoi(string(permStr[2]))

	if inode.I_uid == uid {
		return (userPerm & 1) != 0 // bit 1 = ejecución
	}
	if inode.I_gid == gid {
		return (groupPerm & 1) != 0
	}
	return (otherPerm & 1) != 0
}*/
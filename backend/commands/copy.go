package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"time"

	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
)

// ExecuteCopy: copia archivo o carpeta (recursivo).
func ExecuteCopy(srcPath string, destPath string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: debes iniciar sesión para usar copy.")
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
	srcInode, _, err := fs.FindInodeByPath(file, sb, srcPath)
	if err != nil {
		fmt.Println("Error: no se encontró la ruta origen:", srcPath)
		return
	}

	if !tienePermisoLectura(srcInode, uid, gid) {
		fmt.Println("Error: no tienes permiso de lectura sobre el origen.")
		return
	}

	// --- Determinar destino real ---
	// Si destPath existe y es carpeta -> usarla como padre y nombre = base(srcPath)
	// Si destPath existe y es archivo -> usar ese archivo (se sobreescribirá)
	// Si destPath no existe -> buscar padre y usar base(destPath) como nombre de archivo
	var parentInode structs.Inode
	var parentInodeIndex int32
	var destName string
	//destIsDir := false
	//destIsDir := false

	// Intentar encontrar destPath tal cual
	destInode, destIdx, errDest := fs.FindInodeByPath(file, sb, destPath)
	if errDest == nil {
		// encontrado
		if destInode.I_type == 0 {
			// es carpeta: copiar dentro con mismo nombre de origen
			parentInode = destInode
			parentInodeIndex = destIdx
			destName = path.Base(srcPath)
			//destIsDir = true
		} else {
			// es archivo: usaremos su padre y su nombre (sobreescribir)
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
			parentInode = pInode
			parentInodeIndex = pIdx
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
		parentInode = pInode
		parentInodeIndex = pIdx
		destName = path.Base(destPath)
	}

	// Permiso escritura en carpeta destino
	if !tienePermisoEscritura(parentInode, uid, gid) {
		fmt.Println("Error: no tienes permiso de escritura en la carpeta destino.")
		return
	}

	// --- Copiar recursivamente ---
	if srcInode.I_type == 1 {
		// archivo
		data, err := fs.ReadFileContent(file, sb, srcInode)
		if err != nil {
			fmt.Println("Error al leer el archivo origen:", err)
			return
		}
		if err := writeFileToParent(file, sb, parentInodeIndex, destName, data, uid, gid); err != nil {
			fmt.Println("Error al escribir archivo destino:", err)
			return
		}
	} else {
		// carpeta
		//newFolderPath := path.Join(path.Clean("/"), path.Join(path.Clean(path.Dir(destPath)), destName))
		// Crear carpeta destino (usa ExecuteMkdir para reutilizar código de creación de carpetas)
		ExecuteMkdir(path.Join(path.Dir(destPath), destName), true)
		// Leer de nuevo para obtener el nuevo inodo de la carpeta creada
		newParentInode, _, err := fs.FindInodeByPath(file, sb, path.Join(path.Dir(destPath), destName))
		if err != nil {
			fmt.Println("Error al obtener carpeta destino creada:", err)
			return
		}
		// Recorrer contenido fuente
		copyFolderRecursive(file, sb, srcInode, newParentInode, path.Join(path.Dir(destPath), destName), uid, gid)
	}

	fmt.Println("Copia completada correctamente.")
	addJournalEntry(file, sb, mountedPartition.Start, "COPY", srcPath+" -> "+destPath, "-")
}

// writeFileToParent escribe bytes en la carpeta padre (crea o sobreescribe archivo).
func writeFileToParent(file *os.File, sb structs.Superblock, parentInodeIndex int32, fileName string, data []byte, uid, gid int32) error {
	// Leer parent inode
	parentInode, err := fs.ReadInode(file, sb, parentInodeIndex)
	if err != nil {
		return fmt.Errorf("error leyendo inodo padre: %v", err)
	}

	// Buscar entrada existente
	var existingInodeIndex int32 = -1
	//var foundBlockIdx int32 = -1
	//var foundEntryIdx int = -1

	for _, blockNum := range parentInode.I_block {
		if blockNum == -1 {
			continue
		}
		fb, err := fs.ReadFolderBlock(file, sb, blockNum)
		if err != nil {
			continue
		}
		for _, entry := range fb.B_content {			
			name := string(bytes.Trim(entry.B_name[:], "\x00"))
			if name == fileName && entry.B_inodo != -1 {
				existingInodeIndex = entry.B_inodo
				//foundBlockIdx = blockNum
				//foundEntryIdx = idx
				break
			}
		}
		if existingInodeIndex != -1 {
			break
		}
	}

	// Si existe, sobrescribir su inodo
	if existingInodeIndex != -1 {
		inode, err := fs.ReadInode(file, sb, existingInodeIndex)
		if err != nil {
			return fmt.Errorf("error leyendo inodo existente: %v", err)
		}
		// Limpiar punteros actuales (no implementamos liberación de bloques aquí; simplemente sobrescribimos)
		for i := range inode.I_block {
			inode.I_block[i] = -1
		}
		// Escribir datos en nuevos bloques
		blockSize := int(sb.S_block_size)
		offset := 0
		for i := 0; i < len(inode.I_block) && offset < len(data); i++ {
			bIdx, _ := fs.FindFreeBlock(file, sb)
			fs.MarkBlockAsUsed(file, sb, bIdx)
			end := offset + blockSize
			if end > len(data) {
				end = len(data)
			}
			var fb structs.FileBlock
			copy(fb.B_content[:], data[offset:end])
			if err := fs.WriteFileBlock(file, sb, bIdx, fb); err != nil {
				return fmt.Errorf("error escribiendo bloque: %v", err)
			}
			inode.I_block[i] = bIdx
			offset = end
		}
		inode.I_size = int32(len(data))
		inode.I_mtime = time.Now().Unix()
		inode.I_uid = uid
		inode.I_gid = gid
		if err := fs.WriteInode(file, sb, existingInodeIndex, inode); err != nil {
			return fmt.Errorf("error actualizando inodo: %v", err)
		}
		return nil
	}

	// Si no existe: crear nuevo inodo y añadir entrada en carpeta padre
	newInodeIndex, _ := fs.FindFreeInode(file, sb)
	fs.MarkInodeAsUsed(file, sb, newInodeIndex)

	var newInode structs.Inode
	newInode.I_uid = uid
	newInode.I_gid = gid
	newInode.I_type = 1
	newInode.I_perm = 664
	newInode.I_atime = time.Now().Unix()
	newInode.I_ctime = time.Now().Unix()
	newInode.I_mtime = time.Now().Unix()
	for i := range newInode.I_block {
		newInode.I_block[i] = -1
	}

	// Escribir contenido en bloques
	blockSize := int(sb.S_block_size)
	offset := 0
	for i := 0; offset < len(data) && i < len(newInode.I_block); i++ {
		blockIndex, _ := fs.FindFreeBlock(file, sb)
		fs.MarkBlockAsUsed(file, sb, blockIndex)

		end := offset + blockSize
		if end > len(data) {
			end = len(data)
		}

		var fb structs.FileBlock
		copy(fb.B_content[:], data[offset:end])
		offset = end

		newInode.I_block[i] = blockIndex
		if err := fs.WriteFileBlock(file, sb, blockIndex, fb); err != nil {
			return fmt.Errorf("error escribiendo bloque: %v", err)
		}
	}
	newInode.I_size = int32(len(data))

	if err := fs.WriteInode(file, sb, newInodeIndex, newInode); err != nil {
		return fmt.Errorf("error escribiendo inodo: %v", err)
	}

	// Insertar entrada en carpeta padre
	inserted := false
	for _, blockNum := range parentInode.I_block {
		if blockNum == -1 {
			continue
		}
		parentFB, _ := fs.ReadFolderBlock(file, sb, blockNum)
		for idx, entry := range parentFB.B_content {
			if entry.B_inodo == -1 {
				copy(parentFB.B_content[idx].B_name[:], []byte(fileName))
				parentFB.B_content[idx].B_inodo = newInodeIndex
				if err := fs.WriteFolderBlock(file, sb, blockNum, parentFB); err != nil {
					return fmt.Errorf("error escribiendo bloque carpeta padre: %v", err)
				}
				inserted = true
				break
			}
		}
		if inserted {
			break
		}
	}

	if !inserted {
		// Crear nuevo bloque de carpeta
		newParentBlockIndex, _ := fs.FindFreeBlock(file, sb)
		fs.MarkBlockAsUsed(file, sb, newParentBlockIndex)

		var parentFB structs.FolderBlock
		for i := range parentFB.B_content {
			parentFB.B_content[i].B_inodo = -1
		}
		copy(parentFB.B_content[0].B_name[:], []byte(fileName))
		parentFB.B_content[0].B_inodo = newInodeIndex
		if err := fs.WriteFolderBlock(file, sb, newParentBlockIndex, parentFB); err != nil {
			return fmt.Errorf("error escribiendo nuevo bloque carpeta padre: %v", err)
		}

		wrote := false
		for i := range parentInode.I_block {
			if parentInode.I_block[i] == -1 {
				parentInode.I_block[i] = newParentBlockIndex
				if err := fs.WriteInode(file, sb, parentInodeIndex, parentInode); err != nil {
					return fmt.Errorf("error actualizando inodo padre: %v", err)
				}
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

// copyFolderRecursive copia el contenido de una carpeta fuente dentro de la carpeta destino (ya creada).
func copyFolderRecursive(file *os.File, sb structs.Superblock, srcInode structs.Inode, destInode structs.Inode, destPath string, uid, gid int32) {
	// Leer cada entrada del directorio fuente y copiar según tipo
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
			if entryInode.I_type == 1 {
				// archivo
				data, err := fs.ReadFileContent(file, sb, entryInode)
				if err != nil {
					fmt.Println("Error leyendo archivo fuente:", err)
					continue
				}
				if err := writeFileToParent(file, sb, destInode.I_block[0], entryName, data, uid, gid); err != nil {
					// Nota: intenta encontrar el índice del inodo destino carpeta si I_block[0] no es el bloque que contiene la entrada
					// Para simplicidad se asume que destInode está sincronizado y writeFileToParent busca en todos sus bloques.
					fmt.Println("Error escribiendo archivo destino:", err)
				}
			} else {
				// carpeta: crear y recursar
				newDestPath := path.Join(destPath, entryName)
				ExecuteMkdir(newDestPath, true)
				newDestInode, _, err := fs.FindInodeByPath(file, sb, newDestPath)
				if err != nil {
					fmt.Println("Error al obtener carpeta destino creada:", err)
					continue
				}
				copyFolderRecursive(file, sb, entryInode, newDestInode, newDestPath, uid, gid)
			}
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

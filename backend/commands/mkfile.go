package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"time"
	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
)

func ExecuteMkfile(path string, r bool, size int, cont string) {
	if size < 0 {
		fmt.Println("Error: el tamaño no puede ser negativo")
		return
	}

	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar mkfile.")
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
		fmt.Println("Error: No se encontró la partición activa.")
		return
	}

	file, _ := os.OpenFile(mountedPartition.Path, os.O_RDWR, 0644)
	defer file.Close()

	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	binary.Read(file, binary.BigEndian, &sb)

	uid, gid, _ := getUserIDs(file, sb, state.CurrentSession.User)

	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) == 0 {
		fmt.Println("Error: ruta inválida")
		return
	}

	currentInodeIndex := int32(0)

	// Crear carpetas padre si es necesario
	for i, part := range pathParts[:len(pathParts)-1] {
		inode, _ := fs.ReadInode(file, sb, currentInodeIndex)
		found := false

		for _, blockNum := range inode.I_block {
			if blockNum == -1 {
				continue
			}
			fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
			for _, entry := range fb.B_content {
				name := string(bytes.Trim(entry.B_name[:], "\x00"))
				if name == part && entry.B_inodo != -1 {
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
			if r {
				// Crear la carpeta padre usando mkdir
				ExecuteMkdir(strings.Join(pathParts[:i+1], "/"), true)
				// Recalcular el currentInodeIndex después de crear la carpeta
				currentInodeIndex = 0
				for _, p := range pathParts[:i+1] {
					tmpInode, _ := fs.ReadInode(file, sb, currentInodeIndex)
					tmpFound := false
					for _, blockNum := range tmpInode.I_block {
						if blockNum == -1 {
							continue
						}
						fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
						for _, entry := range fb.B_content {
							name := string(bytes.Trim(entry.B_name[:], "\x00"))
							if name == p && entry.B_inodo != -1 {
								currentInodeIndex = entry.B_inodo
								tmpFound = true
								break
							}
						}
						if tmpFound {
							break
						}
					}
				}
			} else {
				fmt.Println("Error: carpeta padre no existe y -r no fue usado")
				return
			}
		}
	}

	parentInode, _ := fs.ReadInode(file, sb, currentInodeIndex)
	if !tienePermisoEscritura(parentInode, uid, gid) {
		fmt.Println("Error: no tienes permiso de escritura en la carpeta padre")
		return
	}

	fileName := pathParts[len(pathParts)-1]

	// Crear el inodo del archivo
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

	// Preparar contenido
	var content []byte
	if cont != "" {
		// Leer archivo real desde la PC
		fileContent, err := os.ReadFile(cont)
		if err != nil {
			fmt.Println("Error: no se pudo leer el archivo de origen:", err)
			return
		}
		content = fileContent
		newInode.I_size = int32(len(content))
	} else {
		content = make([]byte, size)
		for i := 0; i < size; i++ {
			content[i] = byte('0' + i%10)
		}
		newInode.I_size = int32(size)
	}

	// Escribir contenido en bloques
	blockSize := len(structs.FileBlock{}.B_content)
	offset := 0
	for i := 0; offset < len(content) && i < len(newInode.I_block); i++ {
		blockIndex, _ := fs.FindFreeBlock(file, sb)
		fs.MarkBlockAsUsed(file, sb, blockIndex)

		end := offset + blockSize
		if end > len(content) {
			end = len(content)
		}

		var fb structs.FileBlock
		copy(fb.B_content[:], content[offset:end])
		offset = end

		newInode.I_block[i] = blockIndex
		fs.WriteFileBlock(file, sb, blockIndex, fb)
	}

	fs.WriteInode(file, sb, newInodeIndex, newInode)

	// Actualizar carpeta padre (registrar archivo en carpeta)
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
				fs.WriteFolderBlock(file, sb, blockNum, parentFB)
				inserted = true
				break
			}
		}
		if inserted {
			break
		}
	}

	// Si no se encontró espacio, crear un bloque nuevo para la carpeta padre
	if !inserted {
		newParentBlockIndex, _ := fs.FindFreeBlock(file, sb)
		fs.MarkBlockAsUsed(file, sb, newParentBlockIndex)

		var parentFB structs.FolderBlock
		for i := range parentFB.B_content {
			parentFB.B_content[i].B_inodo = -1
		}
		copy(parentFB.B_content[0].B_name[:], []byte(fileName))
		parentFB.B_content[0].B_inodo = newInodeIndex
		fs.WriteFolderBlock(file, sb, newParentBlockIndex, parentFB)

		for i := range parentInode.I_block {
			if parentInode.I_block[i] == -1 {
				parentInode.I_block[i] = newParentBlockIndex
				fs.WriteInode(file, sb, currentInodeIndex, parentInode)
				break
			}
		}
	}

	fmt.Println("Archivo creado correctamente:", path)
}

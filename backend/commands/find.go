package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
)

func ExecuteFind(startPath string, namePattern string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: debes iniciar sesión para usar find.")
		return
	}

	// --- 1. Obtener partición activa ---
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

	file, err := os.OpenFile(mountedPartition.Path, os.O_RDONLY, 0644)
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

	// --- 3. Buscar el inodo de la ruta base ---
	startInodeIndex := int32(0)
	if strings.Trim(startPath, "/") != "" {
		parts := strings.Split(strings.Trim(startPath, "/"), "/")
		for _, part := range parts {
			inode, _ := fs.ReadInode(file, sb, startInodeIndex)
			found := false
			for _, blockNum := range inode.I_block {
				if blockNum == -1 {
					continue
				}
				fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
				for _, entry := range fb.B_content {
					name := string(bytes.Trim(entry.B_name[:], "\x00"))
					if name == part && entry.B_inodo != -1 {
						startInodeIndex = entry.B_inodo
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				fmt.Println("Error: la ruta base no existe.")
				return
			}
		}
	}

	startInode, _ := fs.ReadInode(file, sb, startInodeIndex)
	if !tienePermisoLectura(startInode, uid, gid) {
		fmt.Println("Error: no tienes permiso de lectura en la carpeta base.")
		return
	}

	// --- 4. Convertir el patrón en una expresión regular ---
	regexPattern := "^" + strings.ReplaceAll(
		strings.ReplaceAll(regexp.QuoteMeta(namePattern), `\?`, "."),
		`\*`, ".*",
	) + "$"
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		fmt.Println("Error: patrón inválido.")
		return
	}

	// --- 5. Buscar recursivamente ---
	fmt.Println("Resultados de búsqueda:")
	findRecursive(file, sb, startInodeIndex, startPath, re, uid, gid)
}

// --- Función recursiva para recorrer las carpetas ---
func findRecursive(file *os.File, sb structs.Superblock, inodeIndex int32, currentPath string, re *regexp.Regexp, uid, gid int32) {
	inode, _ := fs.ReadInode(file, sb, inodeIndex)
	if !tienePermisoLectura(inode, uid, gid) {
		return
	}

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
			if name == "." || name == ".." || name == "" {
				continue
			}

			fullPath := path.Join(currentPath, name)

			// Comparar el nombre con el patrón
			if re.MatchString(name) {
				fmt.Println(" -", fullPath)
			}

			// Revisar si es carpeta
			childInode, _ := fs.ReadInode(file, sb, entry.B_inodo)
			if childInode.I_type == 0 { // carpeta
				findRecursive(file, sb, entry.B_inodo, fullPath, re, uid, gid)
			}
		}
	}
}

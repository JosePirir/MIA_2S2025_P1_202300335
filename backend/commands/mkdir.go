package commands

import (
	
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"
	"strconv"
	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
)

// ================= Helpers =================

func getUserIDs(file *os.File, sb structs.Superblock, username string) (int32, int32, error) {
    inode, _, err := fs.FindInodeByPath(file, sb, "/users.txt")
    if err != nil {
        return 0, 0, err
    }

    contentBytes, err := fs.ReadFileContent(file, sb, inode)
    if err != nil {
        return 0, 0, err
    }

    lines := strings.Split(string(contentBytes), "\n")

    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }

        parts := strings.Split(line, ",")
        if len(parts) < 4 {
            continue
        }

        if parts[1] == "U" && parts[3] == username {
            var uid int32
            fmt.Sscanf(parts[0], "%d", &uid)

            groupName := parts[2]
            var gid int32 = -1

            // Buscar el GID correspondiente al grupo
            for _, l := range lines {
                l = strings.TrimSpace(l)
                if l == "" {
                    continue
                }
                p := strings.Split(l, ",")
                if len(p) < 3 {
                    continue
                }
                if p[1] == "G" && p[2] == groupName {
                    fmt.Sscanf(p[0], "%d", &gid)
                    break
                }
            }

            if gid == -1 {
                return 0, 0, errors.New("grupo del usuario no encontrado")
            }

            return uid, gid, nil
        }
    }

    return 0, 0, errors.New("usuario no encontrado")
}


func tienePermisoEscritura(inode structs.Inode, uid, gid int32) bool {
	//fmt.Println("DEBUG: inode.I_uid =", inode.I_uid, "inode.I_gid =", inode.I_gid, "inode.I_perm =", inode.I_perm, "Current uid =", uid, "gid =", gid)
	ownerPerm := (inode.I_perm / 100) % 10
	groupPerm := (inode.I_perm / 10) % 10
	otherPerm := inode.I_perm % 10

	if uid == inode.I_uid {
		return ownerPerm&2 != 0
	}
	if gid == inode.I_gid {
		return groupPerm&2 != 0
	}
	return otherPerm&2 != 0
}

func tienePermisoLectura(inode structs.Inode, uid int32, gid int32) bool {
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
}

// ================= Ejecutables =================

func ExecuteMkdir(path string, p bool) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar mkdir.")
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

	file, err := os.OpenFile(mountedPartition.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer superbloque:", err)
		return
	}

	uid, gid, err := getUserIDs(file, sb, state.CurrentSession.User)
	if err != nil {
		fmt.Println("Error al obtener UID/GID:", err)
		return
	}

	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) == 0 {
		fmt.Println("Error: ruta inválida")
		return
	}

	currentInodeIndex := int32(0) // raíz

	for i, part := range pathParts {
		inode, err := fs.ReadInode(file, sb, currentInodeIndex)
		if err != nil {
			fmt.Println("Error al leer inodo:", err)
			return
		}
		if inode.I_type != 0 {
			fmt.Println("Error: parte intermedia no es carpeta")
			return
		}

		found := false
		for _, blockNum := range inode.I_block {
			if blockNum == -1 {
				continue
			}
			fb, _ := fs.ReadFolderBlock(file, sb, blockNum)
			for _, entry := range fb.B_content {
				name := strings.Trim(string(entry.B_name[:]), "\x00 ")
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
			if i < len(pathParts)-1 && !p {
				// Es carpeta intermedia y -p no fue usado
				fmt.Println("Error: la carpeta padre", part, "no existe y -p no fue usado")
				return
			}

			// Crear carpeta
			if !tienePermisoEscritura(inode, uid, gid) {
				fmt.Println("Error: no tienes permiso de escritura en la carpeta padre")
				return
			}

			newInodeIndex, _ := fs.FindFreeInode(file, sb)
			newBlockIndex, _ := fs.FindFreeBlock(file, sb)
			fs.MarkInodeAsUsed(file, sb, newInodeIndex)
			fs.MarkBlockAsUsed(file, sb, newBlockIndex)

			var newInode structs.Inode
			newInode.I_uid = uid
			newInode.I_gid = gid
			newInode.I_size = int32(unsafe.Sizeof(structs.FolderBlock{}))
			newInode.I_atime = time.Now().Unix()
			newInode.I_ctime = time.Now().Unix()
			newInode.I_mtime = time.Now().Unix()
			newInode.I_type = 0 // carpeta
			newInode.I_perm = 664
			for j := range newInode.I_block {
				newInode.I_block[j] = -1
			}
			newInode.I_block[0] = newBlockIndex
			fs.WriteInode(file, sb, newInodeIndex, newInode)

			var newFolderBlock structs.FolderBlock
			for k := range newFolderBlock.B_content {
				newFolderBlock.B_content[k].B_inodo = -1
			}
			copy(newFolderBlock.B_content[0].B_name[:], []byte("."))
			newFolderBlock.B_content[0].B_inodo = newInodeIndex
			copy(newFolderBlock.B_content[1].B_name[:], []byte(".."))
			newFolderBlock.B_content[1].B_inodo = currentInodeIndex
			fs.WriteFolderBlock(file, sb, newBlockIndex, newFolderBlock)

			// Actualizar carpeta padre (si no hay espacio, asigna un nuevo bloque)
			inserted := false
			for _, blockNum := range inode.I_block {
				if blockNum == -1 {
					// Asignar un nuevo bloque al padre
					newParentBlockIndex, _ := fs.FindFreeBlock(file, sb)
					fs.MarkBlockAsUsed(file, sb, newParentBlockIndex)

					var parentFB structs.FolderBlock
					for z := range parentFB.B_content {
						parentFB.B_content[z].B_inodo = -1
					}
					copy(parentFB.B_content[0].B_name[:], []byte(part))
					parentFB.B_content[0].B_inodo = newInodeIndex
					fs.WriteFolderBlock(file, sb, newParentBlockIndex, parentFB)

					// enlazar en el inodo padre
					for j := range inode.I_block {
						if inode.I_block[j] == -1 {
							inode.I_block[j] = newParentBlockIndex
							fs.WriteInode(file, sb, currentInodeIndex, inode)
							break
						}
					}
					inserted = true
					break
				}

				parentFB, _ := fs.ReadFolderBlock(file, sb, blockNum)
				for idx, entry := range parentFB.B_content {
					if entry.B_inodo == -1 {
						copy(parentFB.B_content[idx].B_name[:], []byte(part))
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

			if !inserted {
				fmt.Println("Error: no hay espacio en el padre para registrar la carpeta")
				return
			}

			currentInodeIndex = newInodeIndex
		}
	}

	fmt.Println("Carpeta creada correctamente:", path)
}
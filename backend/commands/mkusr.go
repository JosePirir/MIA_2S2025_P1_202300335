package commands

import (
	"fmt"
	"os"
	"strings"
	"encoding/binary"

	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
)

func ExecuteMkusr(user, password, group string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar mkusr.")
		return
	}

	if state.CurrentSession.User != "root" {
		fmt.Println("Error: Solo el usuario root puede usar mkusr.")
		return
	}

	// Obtener partición activa
	var mountedPartition *state.MountedPartition
	for _, p := range state.GlobalMountedPartitions {
		if p.ID == state.CurrentSession.PartitionID {
			mountedPartition = &p
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
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// Buscar inodo de /users.txt
	inode, inodeIndex, err := fs.FindInodeByPath(file, sb, "/users.txt")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Leer contenido actual
	contentBytes, err := fs.ReadFileContent(file, sb, inode)
	if err != nil {
		fmt.Println("Error al leer /users.txt:", err)
		return
	}
	content := string(contentBytes)

	// Validar existencia del grupo y si el usuario ya existe
	lines := strings.Split(content, "\n")
	groupExists := false
	maxUID := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		if parts[1] == "G" && parts[2] == group && parts[0] != "0" {
			groupExists = true
		}
		if parts[1] == "U" {
			var uid int
			fmt.Sscanf(parts[0], "%d", &uid)
			if uid > maxUID {
				maxUID = uid
			}
			if parts[3] == user && parts[0] != "0" {
				fmt.Printf("Error: El usuario '%s' ya existe.\n", user)
				return
			}
		}
	}

	if !groupExists {
		fmt.Printf("Error: El grupo '%s' no existe.\n", group)
		return
	}

	// Nuevo UID
	newUID := maxUID + 1
	newLine := fmt.Sprintf("%d,U,%s,%s,%s\n", newUID, group, user, password)
	newContent := content + newLine
	data := []byte(newContent)

	// Guardar en bloques (pidiendo nuevos si hacen falta)
	offset := 0
	blockSize := int(sb.S_block_size)
	for i := 0; i < len(inode.I_block) && offset < len(data); i++ {
		if inode.I_block[i] == -1 {
			freeBlock, err := fs.FindFreeBlock(file, sb)
			if err != nil {
				fmt.Println("Error: no hay bloques disponibles")
				return
			}
			if err := fs.MarkBlockAsUsed(file, sb, freeBlock); err != nil {
				fmt.Println("Error al marcar bloque como usado:", err)
				return
			}
			inode.I_block[i] = freeBlock
		}

		end := offset + blockSize
		if end > len(data) {
			end = len(data)
		}

		var block structs.FileBlock
		copy(block.B_content[:], data[offset:end])
		if err := fs.WriteFileBlock(file, sb, inode.I_block[i], block); err != nil {
			fmt.Println("Error al escribir bloque:", err)
			return
		}

		offset = end
	}

	// Actualizar tamaño del archivo en el inodo
	inode.I_size = int32(len(data))
	if err := fs.WriteInode(file, sb, inodeIndex, inode); err != nil {
		fmt.Println("Error al actualizar inodo:", err)
		return
	}

	fmt.Printf("Usuario '%s' creado exitosamente con UID %d en el grupo '%s'\n", user, newUID, group)
}
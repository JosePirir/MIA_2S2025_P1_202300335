package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"proyecto1/state"
	"proyecto1/structs"
	"strings"
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

	// Leer inodo raíz
	var currentInode structs.Inode
	file.Seek(int64(sb.S_inode_start), 0)
	if err := binary.Read(file, binary.BigEndian, &currentInode); err != nil {
		fmt.Println("Error al leer el inodo raíz:", err)
		return
	}

	// Buscar el inodo de /users.txt
	path := "/users.txt"
	parts := strings.Split(path, "/")
	for i := 1; i < len(parts); i++ {
		name := parts[i]
		if name == "" {
			continue
		}

		found := false
		for _, blockNum := range currentInode.I_block {
			if blockNum == -1 {
				continue
			}
			blockPos := int64(sb.S_block_start) + int64(blockNum)*int64(sb.S_block_size)
			var folderBlock structs.FolderBlock
			file.Seek(blockPos, 0)
			if err := binary.Read(file, binary.BigEndian, &folderBlock); err != nil {
				fmt.Println("Error al leer bloque de carpeta:", err)
				return
			}

			for _, entry := range folderBlock.B_content {
				entryName := string(bytes.Trim(entry.B_name[:], "\x00"))
				if entryName == name {
					file.Seek(int64(sb.S_inode_start)+int64(entry.B_inodo)*int64(sb.S_inode_size), 0)
					if err := binary.Read(file, binary.BigEndian, &currentInode); err != nil {
						fmt.Println("Error al leer inodo:", err)
						return
					}
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			fmt.Printf("Error: No se encontró '%s' en la ruta.\n", name)
			return
		}
	}

	// Leer contenido actual de /users.txt
	var content strings.Builder
	for _, blockNum := range currentInode.I_block {
		if blockNum == -1 {
			continue
		}
		blockPos := int64(sb.S_block_start) + int64(blockNum)*int64(sb.S_block_size)
		blockData := make([]byte, sb.S_block_size)
		file.Seek(blockPos, 0)
		if _, err := io.ReadFull(file, blockData); err != nil {
			fmt.Println("Error al leer bloque del archivo:", err)
			return
		}
		content.Write(bytes.Trim(blockData, "\x00"))
	}

	// Validaciones de grupo y usuario
	lines := strings.Split(content.String(), "\n")
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

		if parts[1] == "G" {
			if parts[2] == group && parts[0] != "0" {
				groupExists = true
			}
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

	// Nueva línea
	newLine := fmt.Sprintf("%d,U,%s,%s,%s\n", newUID, group, user, password)
	newContent := content.String() + newLine

	// ⚡ Importante: limpiar bloques antes de escribir
	data := []byte(newContent)
	offset := 0
	for _, blockNum := range currentInode.I_block {
		if blockNum == -1 {
			continue
		}

		blockPos := int64(sb.S_block_start) + int64(blockNum)*int64(sb.S_block_size)
		blockSize := int(sb.S_block_size)

		// Tomamos un pedazo del contenido
		end := offset + blockSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[offset:end]

		// Rellenamos siempre a blockSize (limpiamos basura anterior)
		padded := make([]byte, blockSize)
		copy(padded, chunk)

		file.Seek(blockPos, 0)
		if _, err := file.Write(padded); err != nil {
			fmt.Println("Error al escribir bloque:", err)
			return
		}

		offset += blockSize
	}

	fmt.Printf("Usuario '%s' creado exitosamente con UID %d en el grupo '%s'\n", user, newUID, group)
}


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

func ExecuteRmgrp(groupName string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar rmgroup.")
		return
	}

	if state.CurrentSession.User != "root" {
		fmt.Println("Error: Debes iniciar sesión con un usuario para usar mkgrp.")
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

	// Buscar el inodo del archivo /users.txt
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

	// Leer contenido actual
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

	// Procesar líneas, marcando como eliminado
	lines := strings.Split(content.String(), "\n")
	removed := false

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) >= 3 && parts[1] == "G" {
			if parts[2] == groupName {
				// Marcar como eliminado (ID -> 0)
				lines[i] = fmt.Sprintf("0,G,%s", parts[2])
				removed = true
			}
		}
	}

	if !removed {
		fmt.Printf("Error: No se encontró el grupo '%s'.\n", groupName)
		return
	}

	newContent := strings.Join(lines, "\n")
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	// Escribir de nuevo en bloques
	data := []byte(newContent)
	offset := 0
	for _, blockNum := range currentInode.I_block {
		if blockNum == -1 {
			continue
		}
		blockPos := int64(sb.S_block_start) + int64(blockNum)*int64(sb.S_block_size)
		blockSize := int(sb.S_block_size)

		end := offset + blockSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[offset:end]

		// Rellenar con ceros
		if len(chunk) < blockSize {
			padded := make([]byte, blockSize)
			copy(padded, chunk)
			chunk = padded
		}

		file.Seek(blockPos, 0)
		if _, err := file.Write(chunk); err != nil {
			fmt.Println("Error al escribir bloque:", err)
			return
		}

		offset += blockSize
		if offset >= len(data) {
			break
		}
	}

	fmt.Printf("Grupo '%s' marcado como eliminado en /users.txt\n", groupName)
}

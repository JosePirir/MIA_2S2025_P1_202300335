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

func ExecuteMkgrp(name string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar mkgrp.")
		return
	}

	if state.CurrentSession.User != "root" {
		fmt.Println("Error: Solo el usuario root puede usar mkgrp.")
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
	_, err = file.Seek(mountedPartition.Start, 0)
	if err != nil {
		fmt.Println("Error al posicionar en superbloque:", err)
		return
	}
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// Buscar el inodo del archivo /users.txt
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

	// Validar si el grupo ya existe
	lines := strings.Split(content, "\n")
	maxID := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) >= 3 && parts[1] == "G" {
			var id int
			fmt.Sscanf(parts[0], "%d", &id)
			if id > maxID {
				maxID = id
			}
			if parts[2] == name && parts[0] != "0" {
				fmt.Printf("Error: El grupo '%s' ya existe.\n", name)
				return
			}
		}
	}

	// Generar ID nuevo
	newID := maxID + 1
	newLine := fmt.Sprintf("%d,G,%s\n", newID, name)
	newContent := content + newLine
	data := []byte(newContent)

	// Guardar el nuevo contenido en bloques
	blockSize := int(sb.S_block_size)
	offset := 0
	for i := 0; i < 12 && offset < len(data); i++ {
		if inode.I_block[i] == -1 {
			// No hay bloque asignado → pedir uno nuevo
			freeBlock, err := fs.FindFreeBlock(file, sb)
			if err != nil {
				fmt.Println("Error: no hay bloques disponibles")
				return
			}
			err = fs.MarkBlockAsUsed(file, sb, freeBlock)
			if err != nil {
				fmt.Println("Error al marcar bloque como usado:", err)
				return
			}
			inode.I_block[i] = freeBlock
		}

		// Crear bloque con los siguientes datos
		var block structs.FileBlock
		end := offset + blockSize
		if end > len(data) {
			end = len(data)
		}
		copy(block.B_content[:], data[offset:end])

		// Guardar bloque en disco
		err = fs.WriteFileBlock(file, sb, inode.I_block[i], block)
		if err != nil {
			fmt.Println("Error al escribir bloque:", err)
			return
		}

		offset = end
	}

	// Actualizar tamaño del archivo en el inodo
	inode.I_size = int32(len(data))
	err = fs.WriteInode(file, sb, inodeIndex, inode)
	if err != nil {
		fmt.Println("Error al actualizar inodo:", err)
		return
	}

	fmt.Printf("Grupo '%s' agregado exitosamente con ID %d en /users.txt\n", name, newID)
}

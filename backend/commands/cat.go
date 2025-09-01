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

func ExecuteCat(path string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar cat.")
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

	file, err := os.OpenFile(mountedPartition.Path, os.O_RDONLY, 0644)
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

	// Dividir la ruta por "/"
	parts := strings.Split(path, "/")
	for i := 1; i < len(parts); i++ { // saltamos el primer "" de la raíz
		name := parts[i]
		if name == "" {
			continue
		}

		// Leer el bloque de carpeta del inodo actual
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

			// Buscar la entrada que coincida con el nombre
			for _, entry := range folderBlock.B_content {
				entryName := string(bytes.Trim(entry.B_name[:], "\x00"))
				if entryName == name {
					// Leer inodo del archivo o carpeta
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

	// Leer contenido del archivo
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

	fmt.Println(content.String())
}

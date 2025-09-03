package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"time"
	"proyecto1/state"
	"proyecto1/structs"
	"strings"

	"github.com/fogleman/gg"
)

// LSReport genera un reporte tipo 'ls' de una ruta en la partición
func LS(partitionID, imagePath, pathFileLS string) {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: Debes iniciar sesión para usar esta función.")
		return
	}

	// Buscar partición montada
	var mountedPartition *state.MountedPartition
	for _, p := range state.GlobalMountedPartitions {
		if p.ID == partitionID {
			mountedPartition = &p
			break
		}
	}
	if mountedPartition == nil {
		fmt.Println("Error: No se encontró la partición activa.")
		return
	}

	// Abrir disco
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

	// Navegar por la ruta indicada
	parts := strings.Split(pathFileLS, "/")
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

	// Preparar imagen
	const W = 1000
	const H = 600
	dc := gg.NewContext(W, H)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	if err := dc.LoadFontFace("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 14); err != nil {
		fmt.Println("Error al cargar la fuente:", err)
		return
	}

	y := 20
	// Encabezado
	dc.SetRGB(0.2, 0.4, 0.6)
	dc.DrawRectangle(0, float64(y), W, 40)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored("REPORTE LS", W/2, float64(y)+20, 0.5, 0.5)
	y += 60

	// Encabezado de tabla
	headers := []string{"Permisos", "Propietario", "Grupo", "Fecha Mod.", "Hora Mod.", "Tipo", "Fecha Creación", "Nombre"}
	xPositions := []float64{50, 150, 250, 350, 450, 550, 650, 800}

	dc.SetRGB(0.3, 0.3, 0.3)
	dc.DrawRectangle(0, float64(y), W, 30)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	for i, h := range headers {
		dc.DrawStringAnchored(h, xPositions[i], float64(y)+15, 0, 0.5)
	}
	y += 30

	// Recorrer bloques de carpeta del inodo actual
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
			if entryName == "" {
				continue
			}

			var entryInode structs.Inode
			file.Seek(int64(sb.S_inode_start)+int64(entry.B_inodo)*int64(sb.S_inode_size), 0)
			if err := binary.Read(file, binary.BigEndian, &entryInode); err != nil {
				fmt.Println("Error al leer inodo:", err)
				continue
			}

			perm := fmt.Sprintf("%d", entryInode.I_perm)
			prop := fmt.Sprintf("%d", entryInode.I_uid)
			group := fmt.Sprintf("%d", entryInode.I_gid)
			modDate := time.Unix(entryInode.I_mtime, 0).Format("2006-01-02")
			modTime := time.Unix(entryInode.I_mtime, 0).Format("15:04:05")
			tipo := "Archivo"
			if entryInode.I_type == '0' {
				tipo = "Carpeta"
			}
			createDate := time.Unix(entryInode.I_ctime, 0).Format("2006-01-02")

			// Filas alternadas
			if y%2 == 0 {
				dc.SetRGB(0.95, 0.95, 0.95)
			} else {
				dc.SetRGB(0.85, 0.85, 0.85)
			}
			dc.DrawRectangle(0, float64(y), W, 25)
			dc.Fill()
			dc.SetRGB(0, 0, 0)

			values := []string{perm, prop, group, modDate, modTime, tipo, createDate, entryName}
			for i, val := range values {
				dc.DrawStringAnchored(val, xPositions[i], float64(y)+12, 0, 0.5)
			}
			y += 25
		}
	}

	// Guardar imagen
	if err := dc.SavePNG(imagePath); err != nil {
		fmt.Println("Error al guardar la imagen:", err)
		return
	}

	fmt.Println("Reporte LS generado en:", imagePath)
}

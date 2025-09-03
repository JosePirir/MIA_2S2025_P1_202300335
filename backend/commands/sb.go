package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"
	"proyecto1/state"
	"proyecto1/structs"

	"github.com/fogleman/gg"
)

func SB(id string, imagePath string) {
	// Obtener partición montada
	mp, found := state.GetMountedPartitionByID(id)
	if !found {
		fmt.Println("No se encontró la partición con ID:", id)
		return
	}

	// Abrir disco
	file, err := os.Open(mp.Path)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	// Leer superbloque
	var sb structs.Superblock
	_, err = file.Seek(mp.Start, 0)
	if err != nil {
		fmt.Println("Error al posicionarse en la partición:", err)
		return
	}
	err = binary.Read(file, binary.LittleEndian, &sb)
	if err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// Crear imagen
	const W = 800
	const H = 600
	dc := gg.NewContext(W, H)
	dc.SetRGB(1, 1, 1) // fondo blanco
	dc.Clear()

	// Fuente
	if err := dc.LoadFontFace("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 16); err != nil {
		fmt.Println("Error al cargar la fuente:", err)
		return
	}

	y := 20

	// Encabezado principal
	dc.SetRGB(0.2, 0.4, 0.6) // azul
	dc.DrawRectangle(0, float64(y), W, 40)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored("REPORTE DE SUPERBLOQUE", W/2, float64(y)+20, 0.5, 0.5)
	y += 60

	// Encabezado de tabla
	dc.SetRGB(0.3, 0.3, 0.3) // gris oscuro
	dc.DrawRectangle(0, float64(y), W, 30)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored("Campo", 100, float64(y)+15, 0.5, 0.5)
	dc.DrawStringAnchored("Valor", 400, float64(y)+15, 0.5, 0.5)
	y += 30

	// Campos del superbloque
	sbFields := []struct {
		Name  string
		Value string
	}{
		{"S_filesystem_type", fmt.Sprintf("%d", sb.S_filesystem_type)},
		{"S_inodes_count", fmt.Sprintf("%d", sb.S_inodes_count)},
		{"S_blocks_count", fmt.Sprintf("%d", sb.S_blocks_count)},
		{"S_free_blocks_count", fmt.Sprintf("%d", sb.S_free_blocks_count)},
		{"S_free_inodes_count", fmt.Sprintf("%d", sb.S_free_inodes_count)},
		{"S_mtime", time.Unix(sb.S_mtime, 0).Format("2006-01-02 15:04")},
		{"S_umtime", time.Unix(sb.S_umtime, 0).Format("2006-01-02 15:04")},
		{"S_mnt_count", fmt.Sprintf("%d", sb.S_mnt_count)},
		{"S_magic", fmt.Sprintf("0x%X", sb.S_magic)},
		{"S_inode_size", fmt.Sprintf("%d", sb.S_inode_size)},
		{"S_block_size", fmt.Sprintf("%d", sb.S_block_size)},
		{"S_first_ino", fmt.Sprintf("%d", sb.S_first_ino)},
		{"S_first_blo", fmt.Sprintf("%d", sb.S_first_blo)},
		{"S_bm_inode_start", fmt.Sprintf("%d", sb.S_bm_inode_start)},
		{"S_bm_block_start", fmt.Sprintf("%d", sb.S_bm_block_start)},
		{"S_inode_start", fmt.Sprintf("%d", sb.S_inode_start)},
		{"S_block_start", fmt.Sprintf("%d", sb.S_block_start)},
	}

	// Dibujar filas alternadas
	for i, field := range sbFields {
		if i%2 == 0 {
			dc.SetRGB(0.95, 0.95, 0.95) // fila clara
		} else {
			dc.SetRGB(0.85, 0.85, 0.85) // fila más oscura
		}
		dc.DrawRectangle(0, float64(y), W, 25)
		dc.Fill()
		dc.SetRGB(0, 0, 0)
		dc.DrawStringAnchored(field.Name, 100, float64(y)+12, 0.5, 0.5)
		dc.DrawStringAnchored(field.Value, 400, float64(y)+12, 0.5, 0.5)
		y += 25
	}

	// Guardar imagen
	if err := dc.SavePNG(imagePath); err != nil {
		fmt.Println("Error al guardar la imagen:", err)
		return
	}
	fmt.Println("Imagen generada en:", imagePath)
}

package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"
	"proyecto1/state"
	"proyecto1/structs"
	"path/filepath"

	"github.com/fogleman/gg"
)

func DISK(id string, imagePath string) {
	mp, found := state.GetMountedPartitionByID(id)
	if !found {
		fmt.Println("No se encontró el disco con ID:", id)
		return
	}

	file, err := os.Open(mp.Path)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	var mbr structs.MBR
	err = binary.Read(file, binary.LittleEndian, &mbr)
	if err != nil {
		fmt.Println("Error al leer MBR:", err)
		return
	}

	const W = 800
	const H = 1000
	dc := gg.NewContext(W, H)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Fuente
	if err := dc.LoadFontFace("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 16); err != nil {
		fmt.Println("Error al cargar la fuente:", err)
		return
	}

	y := 20

	// Encabezado Disk Report
	dc.SetRGB(0, 0.4, 0.4) // teal oscuro
	dc.DrawRectangle(0, float64(y), W, 30)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored("REPORTE DEL DISCO", W/2, float64(y)+15, 0.5, 0.5)
	y += 40

	// Datos generales del disco (MBR)
	mbrFields := map[string]string{
		"Tamaño del disco": fmt.Sprintf("%d bytes", mbr.Mbr_tamano),
		"Fecha de creación": time.Unix(mbr.Mbr_fecha_creacion, 0).Format("2006-01-02 15:04"),
		"Disk Signature":   fmt.Sprintf("%d", mbr.Mbr_dsk_signature),
	}
	for k, v := range mbrFields {
		dc.SetRGB(0.9, 0.9, 0.9)
		dc.DrawRectangle(0, float64(y), W, 25)
		dc.Fill()
		dc.SetRGB(0, 0, 0)
		dc.DrawStringAnchored(k, 20, float64(y)+12, 0, 0.5)
		dc.DrawStringAnchored(v, 250, float64(y)+12, 0, 0.5)
		y += 25
	}

	// Calcular total ocupado y dibujar barras de particiones
	totalSize := float64(mbr.Mbr_tamano)
	xPos := 0.0
	barHeight := 50.0
	barY := float64(y) + 20

	for _, part := range mbr.Mbr_partitions {
		if part.Part_status != '0' {
			sizePercent := float64(part.Part_s) / totalSize
			width := W * sizePercent

			// Color según tipo
			switch part.Part_type {
			case 'P':
				dc.SetRGB(0.4, 0.6, 1) // azul
			case 'E':
				dc.SetRGB(0.6, 1, 0.6) // verde
			default:
				dc.SetRGB(0.8, 0.8, 0.8) // gris
			}
			dc.DrawRectangle(xPos, barY, width, barHeight)
			dc.Fill()

			// Texto de la partición
			dc.SetRGB(0, 0, 0)
			dc.DrawStringAnchored(string(part.Part_name[:]), xPos+width/2, barY+barHeight/2, 0.5, 0.5)
			dc.DrawStringAnchored(fmt.Sprintf("%.2f%%", sizePercent*100), xPos+width/2, barY+barHeight/2+15, 0.5, 0.5)

			// Avanzar posición
			xPos += width

			// Si es extendida, mostrar EBRs como barras internas
			if part.Part_type == 'E' {
				ebrPos := part.Part_start
				for ebrPos != -1 {
					var ebr structs.EBR
					_, err := file.Seek(ebrPos, 0)
					if err != nil {
						break
					}
					err = binary.Read(file, binary.LittleEndian, &ebr)
					if err != nil {
						break
					}
					if ebr.Part_status != '0' {
						ebrPercent := float64(ebr.Part_s) / totalSize
						width := W * ebrPercent
						dc.SetRGB(1, 0.6, 0.6) // rosa
						dc.DrawRectangle(xPos, barY, width, barHeight/2)
						dc.Fill()
						dc.SetRGB(0, 0, 0)
						dc.DrawStringAnchored(string(ebr.Part_name[:]), xPos+width/2, barY+barHeight/4, 0.5, 0.5)
						dc.DrawStringAnchored(fmt.Sprintf("%.2f%%", ebrPercent*100), xPos+width/2, barY+barHeight/4+12, 0.5, 0.5)

						xPos += width
					}
					ebrPos = ebr.Part_next
				}
			}
		}
	}

	// Espacios libres si los hay
	if xPos < float64(W) {
		freePercent := (float64(W) - xPos) / float64(W)
		dc.SetRGB(0.9, 0.9, 0.9)
		dc.DrawRectangle(xPos, barY, float64(W)-xPos, barHeight)
		dc.Fill()
		dc.SetRGB(0, 0, 0)
		dc.DrawStringAnchored("Libre", xPos+(float64(W)-xPos)/2, barY+barHeight/2, 0.5, 0.5)
		dc.DrawStringAnchored(fmt.Sprintf("%.2f%%", freePercent*100), xPos+(float64(W)-xPos)/2, barY+barHeight/2+15, 0.5, 0.5)
	}

	// Asegurarse de que la carpeta existe antes de guardar
	dir := filepath.Dir(imagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Println("Error al crear las carpetas necesarias:", err)
		return
	}

	// Guardar imagen
	if err := dc.SavePNG(imagePath); err != nil {
		fmt.Println("Error al guardar la imagen:", err)
		return
	}

	fmt.Println("Reporte de disco generado en:", imagePath)
}

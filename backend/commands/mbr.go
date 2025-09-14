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

func MBR(id string, imagePath string) {
	// Función auxiliar para obtener color según tipo de partición
	colorParticion := func(partType byte) (r, g, b float64, nombre string) {
		switch partType {
		case 'P': // Primaria
			return 0.6, 0.8, 1, "Partición Primaria"
		case 'E': // Extendida
			return 0.6, 1, 0.6, "Partición Extendida"
		case 'L': // Lógica
			return 1, 0.8, 0.6, "Partición Lógica"
		default:
			return 0.9, 0.9, 0.9, "Partición Desconocida"
		}
	}

	mp, found := state.GetMountedPartitionByID(id)
	if !found {
		fmt.Println("No se encontró la partición con ID:", id)
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
	const H = 1500
	dc := gg.NewContext(W, H)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Fuente
	if err := dc.LoadFontFace("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 16); err != nil {
		fmt.Println("Error al cargar la fuente:", err)
		return
	}

	y := 20

	// Encabezado MBR
	dc.SetRGB(0.4, 0, 0.4) // morado oscuro
	dc.DrawRectangle(0, float64(y), W, 30)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored("REPORTE DE MBR", W/2, float64(y)+15, 0.5, 0.5)
	y += 40

	// Datos MBR
	mbrFields := map[string]string{
		"mbr_tamano":        fmt.Sprintf("%d", mbr.Mbr_tamano),
		"mbr_fecha_creacion": time.Unix(mbr.Mbr_fecha_creacion, 0).Format("2006-01-02 15:04"),
		"mbr_dsk_signature": fmt.Sprintf("%d", mbr.Mbr_dsk_signature),
	}
	for k, v := range mbrFields {
		dc.SetRGB(0.9, 0.9, 0.9)
		dc.DrawRectangle(0, float64(y), W, 25)
		dc.Fill()
		dc.SetRGB(0, 0, 0)
		dc.DrawStringAnchored(k, 100, float64(y)+12, 0, 0.5)
		dc.DrawStringAnchored(v, 300, float64(y)+12, 0, 0.5)
		y += 25
	}

	// Particiones del MBR
	for _, part := range mbr.Mbr_partitions {
		if part.Part_status != '0' {
			r, g, b, nombre := colorParticion(part.Part_type)
			// Encabezado partición
			dc.SetRGB(r, g, b)
			dc.DrawRectangle(0, float64(y), W, 25)
			dc.Fill()
			dc.SetRGB(0, 0, 0)
			dc.DrawStringAnchored(nombre, 20, float64(y)+12, 0, 0.5)
			y += 25

			partFields := map[string]string{
				"part_status": string(part.Part_status),
				"part_type":   string(part.Part_type),
				"part_fit":    string(part.Part_fit),
				"part_start":  fmt.Sprintf("%d", part.Part_start),
				"part_size":   fmt.Sprintf("%d", part.Part_s),
				"part_name":   string(part.Part_name[:]),
			}
			for k, v := range partFields {
				dc.SetRGB(r, g, b)
				dc.DrawRectangle(0, float64(y), W, 25)
				dc.Fill()
				dc.SetRGB(0, 0, 0)
				dc.DrawStringAnchored(k, 20, float64(y)+12, 0, 0.5)
				dc.DrawStringAnchored(v, 200, float64(y)+12, 0, 0.5)
				y += 25
			}

			// Si es extendida, mostrar EBRs
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
						r, g, b, nombre = colorParticion('L') // lógica
						dc.SetRGB(r, g, b)
						dc.DrawRectangle(0, float64(y), W, 25)
						dc.Fill()
						dc.SetRGB(0, 0, 0)
						dc.DrawStringAnchored(nombre, 20, float64(y)+12, 0, 0.5)
						y += 25

						ebrFields := map[string]string{
							"part_status": string(ebr.Part_status),
							"part_next":   fmt.Sprintf("%d", ebr.Part_next),
							"part_fit":    string(ebr.Part_fit),
							"part_start":  fmt.Sprintf("%d", ebr.Part_start),
							"part_size":   fmt.Sprintf("%d", ebr.Part_s),
							"part_name":   string(ebr.Part_name[:]),
						}
						for k, v := range ebrFields {
							dc.SetRGB(r, g, b)
							dc.DrawRectangle(0, float64(y), W, 25)
							dc.Fill()
							dc.SetRGB(0, 0, 0)
							dc.DrawStringAnchored(k, 20, float64(y)+12, 0, 0.5)
							dc.DrawStringAnchored(v, 200, float64(y)+12, 0, 0.5)
							y += 25
						}
					}
					ebrPos = ebr.Part_next
				}
			}
		}
	}

	// Guardar imagen
	if err := dc.SavePNG(imagePath); err != nil {
		fmt.Println("Error al guardar la imagen:", err)
		return
	}
	fmt.Println("Imagen generada en:", imagePath)
}

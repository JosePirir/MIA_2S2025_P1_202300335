package commands

import (
	"fmt"
	"os"
	"proyecto1/structs"
	"proyecto1/utils"
	"strings"
)

// ExecuteListPartitions lista las particiones (primarias, extendida y lógicas)
// de un disco .mia indicado por path.
func ExecuteListPartitions(path string) {
	if path == "" {
		fmt.Println("Error: se requiere -path")
		return
	}

	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error: no se pudo abrir el disco en '%s': %v\n", path, err)
		return
	}
	defer file.Close()

	mbr, err := utils.ReadMBR(file)
	if err != nil {
		fmt.Printf("Error al leer MBR: %v\n", err)
		return
	}

	// Imprimir particiones primarias / extendida
	for i := range mbr.Mbr_partitions {
		p := mbr.Mbr_partitions[i]
		name := strings.Trim(string(p.Part_name[:]), "\x00")
		if p.Part_s <= 0 {
			continue
		}
		typ := "PRIMARY"
		if p.Part_type == 'E' {
			typ = "EXTENDED"
		}
		// Salida parseable: TYPE|NAME|START|SIZE|STATUS
		fmt.Printf("%s|%s|%d|%d|%c\n", typ, name, p.Part_start, p.Part_s, p.Part_status)
	}

	// Si hay partición extendida, recorrer EBRs y listar lógicas
	var ext structs.Partition
	foundExt := false
	for i := range mbr.Mbr_partitions {
		if mbr.Mbr_partitions[i].Part_type == 'E' {
			ext = mbr.Mbr_partitions[i]
			foundExt = true
			break
		}
	}
	if foundExt {
		ebr, err := utils.ReadEBR(file, ext.Part_start)
		if err != nil {
			// Si no hay EBRs válidos, no hay lógicas
			return
		}
		currentAddr := ext.Part_start
		for {
			// Si EBR tiene tamaño <=0 o name vacío, salir
			name := strings.Trim(string(ebr.Part_name[:]), "\x00")
			if ebr.Part_s > 0 && name != "" {
				fmt.Printf("LOGICAL|%s|%d|%d|%c\n", name, ebr.Part_start, ebr.Part_s, ebr.Part_status)
			}
			if ebr.Part_next == -1 || ebr.Part_next == 0 {
				break
			}
			currentAddr = ebr.Part_next
			ebr, err = utils.ReadEBR(file, currentAddr)
			if err != nil {
				break
			}
		}
	}
}

package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"proyecto1/state"
	"proyecto1/structs"
)

// BM_INODE genera un reporte del bitmap de inodos
func BM_INODE(id, path string) {
	// 1. Buscar la partición montada
	mountedPartition, found := state.GetMountedPartitionByID(id)
	if !found {
		fmt.Printf("Error: No se encontró la partición montada con id '%s'\n", id)
		return
	}

	file, err := os.Open(mountedPartition.Path)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	// 2. Leer el superbloque
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// 3. Leer el bitmap de inodos
	bmInodes := make([]byte, sb.S_inodes_count)
	file.Seek(int64(sb.S_bm_inode_start), 0)
	if err := binary.Read(file, binary.BigEndian, &bmInodes); err != nil {
		fmt.Println("Error al leer el bitmap de inodos:", err)
		return
	}

	// 4. Crear archivo de texto
	reportFile := path + "_bm_inodes.txt"
	f, err := os.Create(reportFile)
	if err != nil {
		fmt.Println("Error al crear el archivo de reporte:", err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "Reporte del Bitmap de Inodos (0 = libre, 1 = ocupado)\n\n")

	// 5. Escribir bits en líneas de 20
	for i := 0; i < len(bmInodes); i++ {
		fmt.Fprintf(f, "%d", bmInodes[i])
		if (i+1)%20 == 0 {
			fmt.Fprintln(f) // salto de línea cada 20 bits
		} else {
			fmt.Fprint(f, " ")
		}
	}

	// Si quedó una línea incompleta, agregar salto de línea
	if len(bmInodes)%20 != 0 {
		fmt.Fprintln(f)
	}

	fmt.Println("Reporte del bitmap de inodos generado en:", reportFile)
}

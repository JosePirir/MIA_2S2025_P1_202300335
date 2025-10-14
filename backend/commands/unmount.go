package commands

import (
	"fmt"
	"os"
	"proyecto1/state"
	"proyecto1/utils"
	"strings"
)

// ExecuteUnmount desmonta una partición del sistema usando su ID.
func ExecuteUnmount(id string) {
	// --- 1. Verificar si la partición existe en la lista global ---
	var targetIndex = -1
	for i, p := range state.GlobalMountedPartitions {
		if p.ID == id {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		fmt.Printf("Error: no se encontró ninguna partición montada con el ID '%s'.\n", id)
		return
	}

	// --- 2. Obtener los datos de la partición a desmontar ---
	mount := state.GlobalMountedPartitions[targetIndex]

	// --- 3. Abrir el archivo del disco ---
	file, err := os.OpenFile(mount.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Printf("Error: no se pudo abrir el disco en '%s'.\n", mount.Path)
		return
	}
	defer file.Close()

	// --- 4. Leer el MBR ---
	mbr, err := utils.ReadMBR(file)
	if err != nil {
		fmt.Printf("Error al leer el MBR: %v\n", err)
		return
	}

	// --- 5. Buscar si la partición es primaria ---
	for i := range mbr.Mbr_partitions {
		p := &mbr.Mbr_partitions[i]
		if strings.Trim(string(p.Part_name[:]), "\x00") == mount.Name {
			// Se encontró la partición primaria a desmontar
			p.Part_status = '0'
			p.Part_correlative = 0

			if err := utils.WriteMBR(file, &mbr); err != nil {
				fmt.Printf("Error al actualizar el MBR en el disco: %v\n", err)
				return
			}

			// Remover de la lista global
			state.GlobalMountedPartitions = append(
				state.GlobalMountedPartitions[:targetIndex],
				state.GlobalMountedPartitions[targetIndex+1:]...,
			)

			fmt.Printf("Partición primaria '%s' desmontada exitosamente (ID: %s).\n", mount.Name, id)
			return
		}
	}

	// --- 6. Buscar si pertenece a una partición lógica ---
	var extendedPartitionFound bool
	var extendedPartitionStart int64

	for i := range mbr.Mbr_partitions {
		if mbr.Mbr_partitions[i].Part_type == 'E' {
			extendedPartitionFound = true
			extendedPartitionStart = mbr.Mbr_partitions[i].Part_start
			break
		}
	}

	if extendedPartitionFound {
		currentEBR, err := utils.ReadEBR(file, extendedPartitionStart)
		if err != nil {
			fmt.Printf("Error al leer el primer EBR: %v\n", err)
			return
		}
		currentAddress := extendedPartitionStart

		for {
			if strings.Trim(string(currentEBR.Part_name[:]), "\x00") == mount.Name {
				currentEBR.Part_status = '0'

				if err := utils.WriteEBR(file, &currentEBR, currentAddress); err != nil {
					fmt.Printf("Error al actualizar el EBR: %v\n", err)
					return
				}

				// Remover de la lista global
				state.GlobalMountedPartitions = append(
					state.GlobalMountedPartitions[:targetIndex],
					state.GlobalMountedPartitions[targetIndex+1:]...,
				)

				fmt.Printf("Partición lógica '%s' desmontada exitosamente (ID: %s).\n", mount.Name, id)
				return
			}

			if currentEBR.Part_next == -1 {
				break
			}

			currentAddress = currentEBR.Part_next
			currentEBR, err = utils.ReadEBR(file, currentAddress)
			if err != nil {
				fmt.Printf("Error al leer la cadena de EBRs: %v\n", err)
				return
			}
		}
	}

	// --- 7. Si no se encontró la partición ---
	fmt.Printf("Error: no se encontró la partición con el nombre '%s' en el disco.\n", mount.Name)
}

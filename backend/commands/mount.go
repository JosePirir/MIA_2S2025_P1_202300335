package commands

import (
	"fmt"
	"os"
	"proyecto1/state"   // Importamos el paquete de estado para acceder a la lista global.
	"proyecto1/structs" // Importamos las estructuras de MBR, EBR, etc.
	"proyecto1/utils"   // Importamos las herramientas para leer/escribir en el disco.
	"strconv"           // Para convertir números a texto (string).
	"strings"
)

// Estas son variables globales para llevar la cuenta de los IDs de montaje.
// Viven en memoria mientras el programa se ejecuta.
var diskLetters = make(map[string]rune)     // Mapa para asignar una letra a cada ruta de disco.
var nextLetter rune = 'A'                   // La siguiente letra disponible para un nuevo disco.
var partitionNumbers = make(map[string]int) // Mapa para llevar el número de la próxima partición por disco.

// ExecuteMount monta una partición en memoria.
func ExecuteMount(path, name string) {
	// --- 1. Verificar si la partición ya está montada ---
	for _, p := range state.GlobalMountedPartitions {
		if p.Path == path && p.Name == name {
			fmt.Printf("Error: la partición '%s' en el disco '%s' ya está montada.\n", name, path)
			return
		}
	}

	// --- 2. Abrir y leer el disco ---
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Printf("Error: no se pudo abrir el disco en '%s'.\n", path)
		return
	}
	defer file.Close()

	mbr, err := utils.ReadMBR(file)
	if err != nil {
		fmt.Printf("Error al leer el MBR: %v\n", err)
		return
	}

	// --- 3. Asignar letra y número base ---
	if _, ok := diskLetters[path]; !ok {
		diskLetters[path] = nextLetter
		nextLetter++
		partitionNumbers[path] = 1
	}

	letter := diskLetters[path]
	partNum := partitionNumbers[path]

	carnet := "202300335"
	id := carnet[len(carnet)-2:] + strconv.Itoa(partNum) + string(letter)

	// --- 4. Buscar particiones primarias ---
	for i := range mbr.Mbr_partitions {
		p := &mbr.Mbr_partitions[i]
		if strings.Trim(string(p.Part_name[:]), "\x00") == name {
			if p.Part_type == 'E' {
				fmt.Println("Error: no se pueden montar particiones extendidas.")
				return
			}

			// Actualiza el disco
			p.Part_status = '1'
			p.Part_correlative = int64(partNum)
			copy(p.Part_id[:], id)

			// Actualiza memoria
			newMount := state.MountedPartition{
				ID:     id,
				Path:   path,
				Name:   name,
				Status: '1',
				Letter: letter,
				PartNum: partNum,
				Size:   p.Part_s,
				Start:  p.Part_start,
			}
			state.GlobalMountedPartitions = append(state.GlobalMountedPartitions, newMount)

			// Incrementa el número SOLO aquí
			partitionNumbers[path]++

			if err := utils.WriteMBR(file, &mbr); err != nil {
				fmt.Printf("Error al actualizar el MBR en el disco: %v\n", err)
				return
			}
			fmt.Printf("Partición primaria '%s' montada exitosamente con el ID: %s\n", name, id)
			return
		}
	}

	// --- 5. Buscar particiones lógicas ---
	var extendedPartition structs.Partition
	foundExtended := false
	for i := range mbr.Mbr_partitions {
		if mbr.Mbr_partitions[i].Part_type == 'E' {
			extendedPartition = mbr.Mbr_partitions[i]
			foundExtended = true
			break
		}
	}

	if foundExtended {
		currentEBR, err := utils.ReadEBR(file, extendedPartition.Part_start)
		if err != nil {
			fmt.Printf("Error al leer el primer EBR: %v\n", err)
			return
		}
		currentEBRAddress := extendedPartition.Part_start

		for {
			if strings.Trim(string(currentEBR.Part_name[:]), "\x00") == name {
				currentEBR.Part_status = '1'

				newMount := state.MountedPartition{
					ID:     id,
					Path:   path,
					Name:   name,
					Status: '1',
					Letter: letter,
					PartNum: partNum,
					Size:   currentEBR.Part_s,
					Start:  currentEBR.Part_start,
				}
				state.GlobalMountedPartitions = append(state.GlobalMountedPartitions, newMount)

				// Incrementa SOLO aquí si se monta
				partitionNumbers[path]++

				if err := utils.WriteEBR(file, &currentEBR, currentEBRAddress); err != nil {
					fmt.Printf("Error al actualizar el EBR en el disco: %v\n", err)
					return
				}
				fmt.Printf("Partición lógica '%s' montada exitosamente con el ID: %s\n", name, id)
				return
			}
			if currentEBR.Part_next == -1 {
				break
			}
			currentEBRAddress = currentEBR.Part_next
			currentEBR, err = utils.ReadEBR(file, currentEBRAddress)
			if err != nil {
				fmt.Printf("Error al leer la cadena de EBRs: %v\n", err)
				return
			}
		}
	}

	// --- 6. Si no encontró la partición ---
	fmt.Printf("Error: no se encontró la partición con el nombre '%s'.\n", name)
}

// ExecuteMounted muestra todas las particiones montadas.
func ExecuteMounted() {
	// Revisa si la lista global está vacía.
	if len(state.GlobalMountedPartitions) == 0 {
		fmt.Println("No hay particiones montadas.")
		return
	}
	// Imprime un encabezado.
	fmt.Println("--- Particiones Montadas ---")
	// Recorre la lista e imprime los datos de cada partición montada.
	for _, p := range state.GlobalMountedPartitions {
		fmt.Printf("- ID: %s, Disco: %s, Partición: %s\n", p.ID, p.Path, p.Name)
	}
	fmt.Println("--------------------------")
}
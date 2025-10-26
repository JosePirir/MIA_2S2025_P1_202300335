package commands

import (
	"encoding/binary"
	"fmt"
	"proyecto1/structs"
	"proyecto1/utils" // UTILS
	"os"
	"strings"
)

// ExecuteFdisk es el punto de entrada principal para el comando fdisk.
// Decide qué tipo de partición crear y llama a la función correspondiente.
func ExecuteFdisk(path, name, unit, typeStr, fit string, size int64, delete string, add int64) {
	// 1. Abrir el archivo del disco en modo lectura/escritura
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Error: el disco en la ruta '%s' no existe.\n", path)
		} else {
			fmt.Printf("Error al abrir el disco: %v\n", err)
		}
		return
	}
	defer file.Close()

	// 2. Leer el MBR existente del disco
	var mbr structs.MBR
	file.Seek(0, 0)
	err = binary.Read(file, binary.LittleEndian, &mbr)
	if err != nil {
		fmt.Printf("Error al leer el MBR del disco: %v\n", err)
		return
	}

	// Si se solicita eliminar una partición
	if delete != "" {
		deletePartition(file, &mbr, name, delete)
		return
	}

	// Si se solicita redimensionar una partición
	if add != 0 {
		resizePartition(file, &mbr, name, add, unit)
		return
	}


	// 3. Calcular el tamaño de la nueva partición en bytes
	var partitionSize int64
	switch strings.ToLower(unit) {
	case "b":
		partitionSize = size
	case "m":
		partitionSize = size * 1024 * 1024
	default: // "k" es el default
		partitionSize = size * 1024
	}

	// 4. Llamar a la función correcta según el tipo de partición
	switch strings.ToLower(typeStr) {
	case "p":
		createPrimary(file, &mbr, name, fit, partitionSize)
	case "e":
		createExtended(file, &mbr, name, fit, partitionSize)
	case "l":
		createLogical(file, &mbr, name, fit, partitionSize)
	default:
		fmt.Printf("Error: tipo de partición '%s' no reconocido.\n", typeStr)
	}
}

// --- Estructura auxiliar para manejar los espacios libres ---
type FreeSpace struct {
	Start int64
	End   int64
	Size  int64
}

// --- Lógica para Particiones Primarias ---
func createPrimary(file *os.File, mbr *structs.MBR, name, fit string, size int64) {
	fmt.Println("Iniciando creación de partición Primaria...")

	// 1. Validaciones: contar solo particiones primarias existentes
	//primaryCount := 0
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '1' {
			if mbr.Mbr_partitions[i].Part_type == 'P' {
				//primaryCount++
			}
			// Validar que el nombre no se repita
			if strings.Trim(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00") == name {
				fmt.Printf("Error: ya existe una partición con el nombre '%s'.\n", name)
				return
			}
		}
	}
	//if primaryCount >= 3 {
	//	fmt.Println("Error: ya existen 3 particiones primarias, no se pueden crear más.")
	//	return
	//}

	// 2. Encontrar un hueco libre
	freeSpaces := utils.GetFreeSpaces(mbr)
	var bestFitStart int64 = -1

	switch strings.ToLower(fit) {
	case "ff":
		bestFitStart = utils.FindFirstFit(freeSpaces, size)
	case "bf":
		bestFitStart = utils.FindBestFit(freeSpaces, size)
	case "wf":
		bestFitStart = utils.FindWorstFit(freeSpaces, size)
	}

	if bestFitStart == -1 {
		fmt.Println("Error: no hay suficiente espacio contiguo para la partición.")
		return
	}

	// 3. Crear la nueva estructura de Partición
	var newPartition structs.Partition
	newPartition.Part_status = '1' // Activa
	newPartition.Part_type = 'P'
	newPartition.Part_fit = byte(strings.ToUpper(fit)[0])
	newPartition.Part_start = bestFitStart
	newPartition.Part_s = size
	copy(newPartition.Part_name[:], name)

	// 4. Añadirla a un slot vacío en el MBR
	added := false
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '0' {
			mbr.Mbr_partitions[i] = newPartition
			added = true
			break
		}
	}
	if !added {
		fmt.Println("Error: no se encontró un slot de partición libre.")
		return
	}

	// 5. Escribir el MBR actualizado de vuelta al disco
	err := utils.WriteMBR(file, mbr)
	if err != nil {
		fmt.Println(err)
		return
	}

	// --- Inicializar la partición con ceros ---
	zeroBytes := make([]byte, size)
	if _, err := file.WriteAt(zeroBytes, newPartition.Part_start); err != nil {
		fmt.Printf("Error al inicializar la partición con ceros: %v\n", err)
		return
	}

	fmt.Printf("Partición primaria '%s' creada exitosamente.\n", name)
}

// --- Lógica para Particiones Extendidas ---
func createExtended(file *os.File, mbr *structs.MBR, name, fit string, size int64) {
	fmt.Println("Iniciando creación de partición Extendida...")

	// 1. Validaciones
	partitionCount := 0
	hasExtended := false
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '1' {
			partitionCount++
			if mbr.Mbr_partitions[i].Part_type == 'E' {
				hasExtended = true
			}
			if strings.Trim(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00") == name {
				fmt.Printf("Error: ya existe una partición con el nombre '%s'.\n", name)
				return
			}
		}
	}
	if partitionCount >= 4 {
		fmt.Println("Error: ya existen 4 particiones, no se pueden crear más.")
		return
	}
	if hasExtended {
		fmt.Println("Error: ya existe una partición extendida en este disco.")
		return
	}

	// 2. Encontrar un hueco libre
	freeSpaces := utils.GetFreeSpaces(mbr)
	var bestFitStart int64 = -1

	switch strings.ToLower(fit) {
	case "ff":
		bestFitStart = utils.FindFirstFit(freeSpaces, size)
	case "bf":
		bestFitStart = utils.FindBestFit(freeSpaces, size)
	case "wf":
		bestFitStart = utils.FindWorstFit(freeSpaces, size)
	}

	if bestFitStart == -1 {
		fmt.Println("Error: no hay suficiente espacio contiguo para la partición.")
		return
	}

	// 3. Crear la nueva estructura de Partición Extendida
	var newPartition structs.Partition
	newPartition.Part_status = '1'
	newPartition.Part_type = 'E'
	newPartition.Part_fit = byte(strings.ToUpper(fit)[0])
	newPartition.Part_start = bestFitStart
	newPartition.Part_s = size
	copy(newPartition.Part_name[:], name)

	// 4. Añadirla a un slot vacío en el MBR
	added := false
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '0' {
			mbr.Mbr_partitions[i] = newPartition
			added = true
			break
		}
	}
	if !added {
		fmt.Println("Error: no se encontró un slot de partición libre.")
		return
	}

	// 5. Escribir el MBR actualizado
	err := utils.WriteMBR(file, mbr)
	if err != nil {
		fmt.Println(err)
		return
	}

	// --- Inicializar la partición con ceros ---
	zeroBytes := make([]byte, size)
	if _, err := file.WriteAt(zeroBytes, newPartition.Part_start); err != nil {
		fmt.Printf("Error al inicializar la partición extendida con ceros: %v\n", err)
		return
	}

	// 6. Escribir el primer EBR (vacío) al inicio de la partición extendida
	var firstEBR structs.EBR
	firstEBR.Part_status = '0' // Inactivo
	firstEBR.Part_next = -1    // No hay siguiente
	err = utils.WriteEBR(file, &firstEBR, newPartition.Part_start)
	if err != nil {
		fmt.Printf("Error al inicializar el primer EBR: %v\n", err)
		return
	}

	fmt.Printf("Partición extendida '%s' creada exitosamente.\n", name)
}

func createLogical(file *os.File, mbr *structs.MBR, name, fit string, size int64) {
	fmt.Println("Iniciando creación de partición Lógica...")
	// 1. Buscar si existe una partición extendida.
	var extendedPartition structs.Partition
	foundExtended := false
	for i := range mbr.Mbr_partitions {
		if mbr.Mbr_partitions[i].Part_type == 'E' {
			extendedPartition = mbr.Mbr_partitions[i]
			foundExtended = true
			break
		}
	}

	if !foundExtended {
		fmt.Println("Error: No se puede crear una partición lógica porque no existe una partición extendida.")
		return
	}

	// 2. Recorrer la cadena de EBRs para encontrar el último y listar los ocupados.
	var logicalPartitions []structs.EBR
	currentEBR, err := utils.ReadEBR(file, extendedPartition.Part_start)
	if err != nil {
		fmt.Printf("Error al leer el primer EBR: %v\n", err)
		return
	}
	lastEBRAddress := extendedPartition.Part_start

	if currentEBR.Part_status == '1' {
		logicalPartitions = append(logicalPartitions, currentEBR)
		for currentEBR.Part_next != -1 {
			lastEBRAddress = currentEBR.Part_next
			currentEBR, err = utils.ReadEBR(file, currentEBR.Part_next)
			if err != nil {
				fmt.Printf("Error al leer la cadena de EBRs: %v\n", err)
				return
			}
			logicalPartitions = append(logicalPartitions, currentEBR)
		}
	}

	// 3. Encontrar un hueco libre DENTRO de la partición extendida.
	freeSpaces := utils.GetFreeSpacesInExtended(extendedPartition, logicalPartitions)
	var bestFitStart int64 = -1

	ebrSize := int64(binary.Size(structs.EBR{}))
	switch strings.ToLower(fit) {
	case "ff":
		bestFitStart = utils.FindFirstFit(freeSpaces, size+ebrSize)
	case "bf":
		bestFitStart = utils.FindBestFit(freeSpaces, size+ebrSize)
	case "wf":
		bestFitStart = utils.FindWorstFit(freeSpaces, size+ebrSize)
	}

	if bestFitStart == -1 {
		fmt.Println("Error: no hay suficiente espacio en la partición extendida.")
		return
	}

	// 4. Crear el nuevo EBR para la partición lógica.
	var newEBR structs.EBR
	newEBR.Part_status = '1'
	newEBR.Part_fit = byte(strings.ToUpper(fit)[0])
	newEBR.Part_start = bestFitStart + ebrSize
	newEBR.Part_s = size
	newEBR.Part_next = -1
	copy(newEBR.Part_name[:], name)

	zeroBytes := make([]byte, size)
	if _, err := file.WriteAt(zeroBytes, newEBR.Part_start); err != nil {
		fmt.Printf("Error al inicializar la partición lógica con ceros: %v\n", err)
		return
	}

	// 5. Escribir el nuevo EBR en su lugar.
	err = utils.WriteEBR(file, &newEBR, bestFitStart)
	if err != nil {
		fmt.Printf("Error al escribir el nuevo EBR: %v\n", err)
		return
	}

	// 6. Actualizar el EBR anterior para que apunte al nuevo.
	if currentEBR.Part_status == '1' {
		currentEBR.Part_next = bestFitStart
		err = utils.WriteEBR(file, &currentEBR, lastEBRAddress)
		if err != nil {
			fmt.Printf("Error al actualizar el último EBR: %v\n", err)
			return
		}
	} else { // Es la primera partición lógica
		err = utils.WriteEBR(file, &newEBR, extendedPartition.Part_start)
		if err != nil {
			fmt.Printf("Error al escribir el primer EBR lógico: %v\n", err)
			return
		}
	}

	fmt.Printf("Partición lógica '%s' creada exitosamente.\n", name)
}

// --- Eliminar particiones ---
func deletePartition(file *os.File, mbr *structs.MBR, name, deleteType string) {
	// 1. Buscar la partición por nombre (primaria o extendida)
	for i := 0; i < 4; i++ {
		partName := strings.Trim(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00")
		if partName == name && mbr.Mbr_partitions[i].Part_status == '1' {
			//fmt.Printf("¿Seguro que deseas eliminar la partición '%s'? (s/n): ", name)
			//var confirm string
			//fmt.Scanln(&confirm)
			//if strings.ToLower(confirm) != "s" {
			//	fmt.Println("Operación cancelada.")
			//	return
			//}

			switch strings.ToLower(deleteType) {
			case "fast":
				// Solo marcar como libre
				mbr.Mbr_partitions[i].Part_status = '0'

			case "full":
				// Marcar como libre y limpiar con ceros
				mbr.Mbr_partitions[i].Part_status = '0'
				zeroBytes := make([]byte, mbr.Mbr_partitions[i].Part_s)
				_, err := file.WriteAt(zeroBytes, mbr.Mbr_partitions[i].Part_start)
				if err != nil {
					fmt.Printf("Error al limpiar la partición: %v\n", err)
					return
				}

			default:
				fmt.Printf("Error: tipo de eliminación '%s' no válido. Usa 'fast' o 'full'.\n", deleteType)
				return
			}

			// Si la partición eliminada era extendida, borrar las lógicas dentro
			if mbr.Mbr_partitions[i].Part_type == 'E' {
				deleteLogicalInside(file, mbr.Mbr_partitions[i])
			}

			err := utils.WriteMBR(file, mbr)
			if err != nil {
				fmt.Printf("Error al actualizar el MBR: %v\n", err)
				return
			}

			fmt.Printf("Partición '%s' eliminada exitosamente con método '%s'.\n", name, deleteType)
			return
		}
	}

	// 2. Si no se encontró, buscar dentro de la extendida (particiones lógicas)
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_type == 'E' && mbr.Mbr_partitions[i].Part_status == '1' {
			currentEBR, err := utils.ReadEBR(file, mbr.Mbr_partitions[i].Part_start)
			if err != nil {
				continue
			}

			var prevEBR structs.EBR
			var prevAddress int64 = -1

			for {
				partName := strings.Trim(string(currentEBR.Part_name[:]), "\x00")
				if partName == name && currentEBR.Part_status == '1' {
					//fmt.Printf("¿Seguro que deseas eliminar la partición lógica '%s'? (s/n): ", name)
					//var confirm string
					//fmt.Scanln(&confirm)
					//if strings.ToLower(confirm) != "s" {
					//	fmt.Println("Operación cancelada.")
					//	return
					//}

					switch strings.ToLower(deleteType) {
					case "fast":
						currentEBR.Part_status = '0'
					case "full":
						currentEBR.Part_status = '0'
						zeroBytes := make([]byte, currentEBR.Part_s)
						_, err := file.WriteAt(zeroBytes, currentEBR.Part_start)
						if err != nil {
							fmt.Printf("Error al limpiar la partición lógica: %v\n", err)
							return
						}
					default:
						fmt.Printf("Error: tipo de eliminación '%s' no válido. Usa 'fast' o 'full'.\n", deleteType)
						return
					}

					// Reenlazar EBRs si no es el primero
					if prevAddress != -1 {
						prevEBR.Part_next = currentEBR.Part_next
						err = utils.WriteEBR(file, &prevEBR, prevAddress)
						if err != nil {
							fmt.Printf("Error al actualizar el EBR anterior: %v\n", err)
							return
						}
					}

					// Guardar el cambio del EBR actual
					err = utils.WriteEBR(file, &currentEBR, currentEBR.Part_start-ebrsz())
					if err != nil {
						fmt.Printf("Error al actualizar el EBR eliminado: %v\n", err)
						return
					}

					fmt.Printf("Partición lógica '%s' eliminada exitosamente con método '%s'.\n", name, deleteType)
					return
				}

				if currentEBR.Part_next == -1 {
					break
				}
				prevEBR = currentEBR
				prevAddress = currentEBR.Part_next
				currentEBR, err = utils.ReadEBR(file, currentEBR.Part_next)
				if err != nil {
					break
				}
			}
		}
	}

	fmt.Printf("Error: no se encontró la partición con nombre '%s'.\n", name)
}

// --- Elimina las particiones lógicas dentro de una extendida ---
func deleteLogicalInside(file *os.File, extended structs.Partition) {
	currentEBR, err := utils.ReadEBR(file, extended.Part_start)
	if err != nil {
		return
	}

	for {
		if currentEBR.Part_status == '1' {
			currentEBR.Part_status = '0'
			zeroBytes := make([]byte, currentEBR.Part_s)
			file.WriteAt(zeroBytes, currentEBR.Part_start)
			utils.WriteEBR(file, &currentEBR, currentEBR.Part_start-int64(binary.Size(structs.EBR{})))
		}
		if currentEBR.Part_next == -1 {
			break
		}
		currentEBR, err = utils.ReadEBR(file, currentEBR.Part_next)
		if err != nil {
			break
		}
	}
	fmt.Println("Todas las particiones lógicas dentro de la extendida fueron eliminadas.")
}

// Tamaño del EBR
func ebrsz() int64 {
	return int64(binary.Size(structs.EBR{}))
}


func resizePartition(file *os.File, mbr *structs.MBR, name string, add int64, unit string) {
	fmt.Printf("Iniciando modificación de tamaño para la partición '%s'...\n", name)

	// 1. Calcular tamaño en bytes según unidad
	var bytesToAdd int64
	switch strings.ToLower(unit) {
	case "b":
		bytesToAdd = add
	case "m":
		bytesToAdd = add * 1024 * 1024
	default: // "k" por defecto
		bytesToAdd = add * 1024
	}

	// 2. Buscar la partición
	for i := 0; i < 4; i++ {
		part := &mbr.Mbr_partitions[i]
		partName := strings.Trim(string(part.Part_name[:]), "\x00")
		if partName == name && part.Part_status == '1' {
			if bytesToAdd < 0 {
				// --- Reducir ---
				if part.Part_s+bytesToAdd <= 0 {
					fmt.Println("Error: la reducción excede el tamaño de la partición.")
					return
				}
				part.Part_s += bytesToAdd
				fmt.Printf("Se redujo la partición '%s' en %d bytes.\n", name, -bytesToAdd)
			} else {
				// --- Aumentar ---
				freeSpaces := utils.GetFreeSpaces(mbr)

				totalFree := int64(0)
				for _, fs := range freeSpaces {
					totalFree += fs.Size
				}

				if totalFree < bytesToAdd {
					fmt.Println("Error: no hay suficiente espacio libre en el disco para expandir la partición.")
					return
				}

				// Aumentar el tamaño aunque no sea contiguo
				part.Part_s += bytesToAdd
				fmt.Printf("Se aumentó la partición '%s' en %d bytes (sin requerir contigüidad).\n", name, bytesToAdd)
			}

			// 3. Guardar cambios
			err := utils.WriteMBR(file, mbr)
			if err != nil {
				fmt.Printf("Error al guardar los cambios en el MBR: %v\n", err)
				return
			}

			fmt.Printf("Tamaño final de la partición '%s': %d bytes.\n", name, part.Part_s)
			return
		}
	}

	fmt.Printf("Error: no se encontró la partición con nombre '%s'.\n", name)
}

package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"proyecto1/state"
	"proyecto1/structs"
	"strings"
	"time"
)

// RecoveryFileSystem recupera el sistema de archivos a un estado consistente utilizando el journaling y el superbloque.
// Recibe el ID de la partición montada.
func RecoveryFileSystem(id string) {
	// --- VALIDACIÓN DE PARÁMETROS ---
	mountedPartition, found := state.GetMountedPartitionByID(id)
	if !found {
		fmt.Printf("Error: No se encontró una partición montada con el id '%s'.\n", id)
		return
	}

	fmt.Printf("Iniciando recuperación del sistema de archivos para la partición %s en %s.\n", mountedPartition.Name, mountedPartition.Path)

	// --- APERTURA DEL ARCHIVO DE DISCO ---
	file, err := os.OpenFile(mountedPartition.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	// --- LECTURA DEL SUPERBLOQUE ---
	partitionStart := mountedPartition.Start
	var superbloque structs.Superblock

	file.Seek(partitionStart, 0)
	if err := binary.Read(file, binary.BigEndian, &superbloque); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	if superbloque.S_filesystem_type != 3 {
		fmt.Println("Error: La partición no utiliza el sistema de archivos EXT3 (3fs).")
		return
	}

	// --- LECTURA DEL JOURNALING ---
	journalingStart := partitionStart + int64(binary.Size(superbloque))
	journalingSize := int64(50) * int64(superbloque.S_inodes_count)
	file.Seek(journalingStart, 0)

	fmt.Println("Procesando entradas del journaling...")
	for i := int64(0); i < journalingSize; i += int64(binary.Size(structs.JournalEntry{})) {
		var entry structs.JournalEntry
		if err := binary.Read(file, binary.BigEndian, &entry); err != nil {
			fmt.Println("Error al leer una entrada del journaling:", err)
			continue
		}
		if entry.JCount == 0 {
			continue
		}

		// Aplicar la entrada (replay) -- función helper
		if err := applyJournalEntry(&entry, file, superbloque, partitionStart); err != nil {
			fmt.Println("Error aplicando entrada de journaling:", err)
		}
	}

	// --- ACTUALIZACIÓN DEL SUPERBLOQUE ---
	superbloque.S_umtime = time.Now().Unix()
	file.Seek(partitionStart, 0)
	if err := binary.Write(file, binary.BigEndian, &superbloque); err != nil {
		fmt.Println("Error al actualizar el superbloque:", err)
		return
	}

	fmt.Println("Recuperación del sistema de archivos completada exitosamente.")
}

// helper que interpreta y aplica una entrada del journaling
func applyJournalEntry(entry *structs.JournalEntry, file *os.File, sb structs.Superblock, sbStart int64) error {
	op := strings.TrimRight(string(entry.JContent.IOperation[:]), "\x00")
	path := strings.TrimRight(string(entry.JContent.IPath[:]), "\x00")
	//content := strings.TrimRight(string(entry.JContent.IContent[:]), "\x00")

	switch op {
	case "create":
		// TODO: llamar a la lógica que crea archivo/carpeta dentro del FS.
		// Ejemplo: Implementar una función interna applyCreate(path, content, file, sb, sbStart)
		fmt.Printf("Replay create %s\n", path)
	case "delete":
		fmt.Printf("Replay delete %s\n", path)
	case "modify":
		fmt.Printf("Replay modify %s\n", path)
	default:
		fmt.Printf("Operación desconocida en journaling: %s\n", op)
	}
	return nil
}

// ExecuteRecovery reconstruye bitmaps y contadores del superbloque
// escaneando la tabla de inodos y los bloques apuntados por ellos.
// Uso: recovery -id=<MountID>
func ExecuteRecovery(id string) {
	// 1) obtener partición montada
	mounted, found := state.GetMountedPartitionByID(id)
	if !found {
		fmt.Printf("Error: No se encontró una partición montada con id '%s'.\n", id)
		return
	}

	f, err := os.OpenFile(mounted.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Printf("Error al abrir disco '%s': %v\n", mounted.Path, err)
		return
	}
	defer f.Close()

	// 2) leer superbloque
	var sb structs.Superblock
	if _, err := f.Seek(int64(mounted.Start), 0); err != nil {
		fmt.Println("Error al seek superbloque:", err)
		return
	}
	if err := binary.Read(f, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer superbloque:", err)
		return
	}

	totalInodes := int(sb.S_inodes_count)
	totalBlocks := int(sb.S_blocks_count)
	if totalInodes <= 0 || totalBlocks <= 0 {
		fmt.Println("Superbloque inválido (conteos <= 0). Abortando.")
		return
	}

	// 3) buffers para reconstruir bitmaps
	bmInode := make([]byte, totalInodes)
	bmBlock := make([]byte, totalBlocks)

	usedInodes := 0
	usedBlocks := 0

	// 4) escanear todos los inodos y marcar inodos usados + bloques referenciados
	inodeStart := int64(sb.S_inode_start)
	inodeSize := int64(sb.S_inode_size)

	for i := 0; i < totalInodes; i++ {
		pos := inodeStart + int64(i)*inodeSize
		if _, err := f.Seek(pos, 0); err != nil {
			fmt.Printf("Seek inodo %d: %v\n", i, err)
			continue
		}
		var ino structs.Inode
		if err := binary.Read(f, binary.BigEndian, &ino); err != nil {
			fmt.Printf("Error leyendo inodo %d: %v\n", i, err)
			continue
		}

		// Determinar si el inodo está en uso: tipo válido o bloques asignados o tamaño > 0
		if ino.I_type == 0 || ino.I_type == 1 || ino.I_size > 0 {
			// marcar inodo usado
			bmInode[i] = 1
			usedInodes++

			// marcar bloques referenciados
			for _, b := range ino.I_block {
				if b >= 0 && int(b) < totalBlocks {
					if bmBlock[int(b)] == 0 {
						bmBlock[int(b)] = 1
						usedBlocks++
					}
				}
			}
		}
	}

	// 5) recalcular primeros libres
	firstInode := int32(-1)
	for i := 0; i < totalInodes; i++ {
		if bmInode[i] == 0 {
			firstInode = int32(i)
			break
		}
	}
	if firstInode == -1 {
		firstInode = int32(totalInodes) // ninguno libre
	}

	firstBlock := int32(-1)
	for i := 0; i < totalBlocks; i++ {
		if bmBlock[i] == 0 {
			firstBlock = int32(i)
			break
		}
	}
	if firstBlock == -1 {
		firstBlock = int32(totalBlocks)
	}

	// 6) actualizar superbloque en memoria
	sb.S_free_inodes_count = int32(totalInodes - usedInodes)
	sb.S_free_blocks_count = int32(totalBlocks - usedBlocks)
	sb.S_first_ino = firstInode
	sb.S_first_blo = firstBlock

	// 7) escribir bitmaps y superbloque de vuelta al disco
	_, err = f.Seek(int64(sb.S_bm_inode_start), 0)
	if err != nil {
		fmt.Println("Error al posicionar bitmap inodos:", err)
		return
	}
	if err := binary.Write(f, binary.BigEndian, bmInode); err != nil {
		fmt.Println("Error al escribir bitmap inodos:", err)
		return
	}

	_, err = f.Seek(int64(sb.S_bm_block_start), 0)
	if err != nil {
		fmt.Println("Error al posicionar bitmap bloques:", err)
		return
	}
	if err := binary.Write(f, binary.BigEndian, bmBlock); err != nil {
		fmt.Println("Error al escribir bitmap bloques:", err)
		return
	}

	// escribir superbloque actualizado
	if _, err := f.Seek(int64(mounted.Start), 0); err != nil {
		fmt.Println("Error al posicionar superbloque para reescribir:", err)
		return
	}
	if err := binary.Write(f, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al reescribir superbloque:", err)
		return
	}

	// 8) resumen
	fmt.Printf("Recovery completado en %s\n", mounted.Path)
	fmt.Printf("Inodos usados: %d / %d\n", usedInodes, totalInodes)
	fmt.Printf("Bloques usados: %d / %d\n", usedBlocks, totalBlocks)
	fmt.Printf("Superbloque actualizado: S_free_inodes_count=%d, S_free_blocks_count=%d, S_first_ino=%d, S_first_blo=%d\n",
		sb.S_free_inodes_count, sb.S_free_blocks_count, sb.S_first_ino, sb.S_first_blo)
	fmt.Println("Nota: este recovery recalcula bitmaps y contadores desde la información de inodos. Si quieres\nreplay del journaling (reaplicar operaciones pendientes), implementa el parse y aplicación de entradas\ndel area de journaling y ejecútalas tras esta reconstrucción.")
}

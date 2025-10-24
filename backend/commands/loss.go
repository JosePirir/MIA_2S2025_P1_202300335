package commands

import (
    "encoding/binary"
    "fmt"
    "os"
    "proyecto1/state"
    "proyecto1/structs"
    "time"
)
// SimulateSystemLoss formatea (pone a cero) las áreas indicadas de la partición
// para simular pérdida/inconsistencia: bitmap inodos, bitmap bloques, área de inodos y área de bloques.
// Parámetro:
// - id: id de la partición montada (obligatorio).
func SimulateSystemLoss(id string) {
    mountedPartition, found := state.GetMountedPartitionByID(id)
    if !found {
        fmt.Printf("Error: No se encontró una partición montada con el id '%s'.\n", id)
        return
    }

    file, err := os.OpenFile(mountedPartition.Path, os.O_RDWR, 0644)
    if err != nil {
        fmt.Println("Error al abrir el disco:", err)
        return
    }
    defer file.Close()

    partitionStart := mountedPartition.Start
    var sb structs.Superblock
    file.Seek(partitionStart, 0)
    if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
        fmt.Println("Error al leer el superbloque:", err)
        return
    }

    // Obtener offsets desde el superbloque (se asumió que contienen offsets absolutos en bytes).
    bmInodeStart := int64(sb.S_bm_inode_start)
    bmBlockStart := int64(sb.S_bm_block_start)
    inodeStart := int64(sb.S_inode_start)
    blockStart := int64(sb.S_block_start)

    // Calcular tamaños en bytes (bitmap: 1 byte por inodo/bloque según mkfs; áreas: count * size).
    bmInodeSize := int64(sb.S_inodes_count)
    bmBlockSize := int64(sb.S_blocks_count)
    inodeAreaSize := int64(sb.S_inodes_count) * int64(sb.S_inode_size)
    blockAreaSize := int64(sb.S_blocks_count) * int64(sb.S_block_size)

    // Helper: escribir ceros en bloques de hasta 1MB para no alocar todo en memoria.
    const chunkSize = 1024 * 1024
    writeZeros := func(offset, length int64) error {
        zero := make([]byte, chunkSize)
        written := int64(0)
        for written < length {
            toWrite := chunkSize
            rem := length - written
            if rem < int64(toWrite) {
                toWrite = int(rem)
            }
            if _, err := file.WriteAt(zero[:toWrite], offset+written); err != nil {
                return err
            }
            written += int64(toWrite)
        }
        return nil
    }

    fmt.Printf("Simulando pérdida en partición %s (%s)...\n", mountedPartition.Name, mountedPartition.Path)

    if bmInodeSize > 0 {
        if err := writeZeros(bmInodeStart, bmInodeSize); err != nil {
            fmt.Println("Error al limpiar bitmap de inodos:", err)
            return
        }
        fmt.Println("Bitmap de inodos limpiado.")
    }

    if bmBlockSize > 0 {
        if err := writeZeros(bmBlockStart, bmBlockSize); err != nil {
            fmt.Println("Error al limpiar bitmap de bloques:", err)
            return
        }
        fmt.Println("Bitmap de bloques limpiado.")
    }

    if inodeAreaSize > 0 {
        if err := writeZeros(inodeStart, inodeAreaSize); err != nil {
            fmt.Println("Error al limpiar área de inodos:", err)
            return
        }
        fmt.Println("Área de inodos limpiada.")
    }

    if blockAreaSize > 0 {
        if err := writeZeros(blockStart, blockAreaSize); err != nil {
            fmt.Println("Error al limpiar área de bloques:", err)
            return
        }
        fmt.Println("Área de bloques limpiada.")
    }

    // Actualizar tiempo de montaje/desmontaje en superbloque (opcional).
    sb.S_umtime = time.Now().Unix()
    file.Seek(partitionStart, 0)
    if err := binary.Write(file, binary.BigEndian, &sb); err != nil {
        fmt.Println("Error al escribir superbloque tras simulación:", err)
        return
    }

    fmt.Println("Simulación de pérdida completada.")
}
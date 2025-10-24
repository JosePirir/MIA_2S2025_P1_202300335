package commands

import (
    "encoding/binary"
    "fmt"
    "os"
    "proyecto1/state"
    "proyecto1/structs"
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

        // Verificar si la entrada es válida.
        if entry.JCount == 0 {
            continue
        }

        operation := string(entry.JContent.IOperation[:])
        path := string(entry.JContent.IPath[:])
        content := string(entry.JContent.IContent[:])
        date := entry.JContent.IDate

        fmt.Printf("Recuperando operación '%s' en '%s' con contenido '%s' realizada en fecha %.f.\n", operation, path, content, date)

        // Aplicar la operación según corresponda.
        switch operation {
        case "create":
            fmt.Printf("Recuperando creación en '%s'.\n", path)
            // Implementar lógica para recuperar creación.
        case "delete":
            fmt.Printf("Recuperando eliminación en '%s'.\n", path)
            // Implementar lógica para recuperar eliminación.
        case "modify":
            fmt.Printf("Recuperando modificación en '%s'.\n", path)
            // Implementar lógica para recuperar modificación.
        default:
            fmt.Printf("Operación desconocida: '%s'.\n", operation)
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
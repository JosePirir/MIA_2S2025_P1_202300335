package commands

import (
    "encoding/binary"
    "fmt"
    "os"
    "strings"
    "proyecto1/fs"
    "proyecto1/state"
    "proyecto1/structs"
)

// ================= Ejecutables =================

func ExecuteChown(path string, recursive bool, newUser string) {
    if !state.CurrentSession.IsActive {
        fmt.Println("Error: Debes iniciar sesión para usar chown.")
        return
    }

    var mountedPartition *state.MountedPartition
    for _, mp := range state.GlobalMountedPartitions {
        if mp.ID == state.CurrentSession.PartitionID {
            mountedPartition = &mp
            break
        }
    }
    if mountedPartition == nil {
        fmt.Println("Error: No se encontró la partición activa.")
        return
    }

    file, err := os.OpenFile(mountedPartition.Path, os.O_RDWR, 0644)
    if err != nil {
        fmt.Println("Error al abrir el disco:", err)
        return
    }
    defer file.Close()

    // Leer superbloque
    var sb structs.Superblock
    file.Seek(mountedPartition.Start, 0)
    if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
        fmt.Println("Error al leer superbloque:", err)
        return
    }

    // Obtener IDs de usuario actual
    currentUID, _, err := getUserIDs(file, sb, state.CurrentSession.User)
    if err != nil {
        fmt.Println("Error al obtener UID/GID del usuario actual:", err)
        return
    }

    // Obtener IDs del nuevo propietario
    newUID, newGID, err := getUserIDs(file, sb, newUser)
    if err != nil {
        fmt.Println("Error: el usuario", newUser, "no existe.")
        return
    }

    // Buscar inodo del archivo o carpeta
    inode, inodeIndex, err := fs.FindInodeByPath(file, sb, path)
    if err != nil {
        fmt.Println("Error: no se encontró la ruta:", path)
        return
    }

    // Verificar permisos
    if state.CurrentSession.User != "root" && inode.I_uid != currentUID {
        fmt.Println("Error: no tienes permisos para cambiar propietario de este archivo.")
        return
    }

    // Cambiar propietario
    inode.I_uid = newUID
    inode.I_gid = newGID
    fs.WriteInode(file, sb, inodeIndex, inode)

    // Si es recursivo y es una carpeta, aplicar a su contenido
    if recursive && inode.I_type == 0 {
        applyChownRecursive(file, sb, inodeIndex, newUID, newGID)
    }

    fmt.Println("Propietario cambiado correctamente a", newUser, "en:", path)
}

// Función recursiva auxiliar
func applyChownRecursive(file *os.File, sb structs.Superblock, inodeIndex, newUID, newGID int32) {
    inode, err := fs.ReadInode(file, sb, inodeIndex)
    if err != nil {
        return
    }

    for _, block := range inode.I_block {
        if block == -1 {
            continue
        }

        fb, err := fs.ReadFolderBlock(file, sb, block)
        if err != nil {
            continue
        }

        for _, entry := range fb.B_content {
            name := strings.Trim(string(entry.B_name[:]), "\x00 ")
            if entry.B_inodo == -1 || name == "." || name == ".." {
                continue
            }

            childInode, err := fs.ReadInode(file, sb, entry.B_inodo)
            if err != nil {
                continue
            }

            // Cambiar propietario
            childInode.I_uid = newUID
            childInode.I_gid = newGID
            fs.WriteInode(file, sb, entry.B_inodo, childInode)

            // Si es carpeta, aplicar recursivamente
            if childInode.I_type == 0 {
                applyChownRecursive(file, sb, entry.B_inodo, newUID, newGID)
            }
        }
    }
}

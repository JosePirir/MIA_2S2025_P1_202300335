package commands

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "os"
    "proyecto1/state"
    "proyecto1/structs"
    "time"
	"strings"
)

// ShowJournal lee el área de journaling de la partición indicada y emite una tabla HTML
// con las columnas: Operacion, Path, Contenido, Fecha.
// Parámetro:
// - id: id de la partición montada (obligatorio).
func ShowJournal(id string) {
    mountedPartition, found := state.GetMountedPartitionByID(id)
    if !found {
        fmt.Printf("Error: No se encontró una partición montada con el id '%s'.\n", id)
        return
    }

    f, err := os.OpenFile(mountedPartition.Path, os.O_RDONLY, 0)
    if err != nil {
        fmt.Println("Error al abrir el disco:", err)
        return
    }
    defer f.Close()

    partitionStart := mountedPartition.Start
    var sb structs.Superblock
    if _, err := f.Seek(partitionStart, 0); err != nil {
        fmt.Println("Error al posicionar el disco:", err)
        return
    }
    if err := binary.Read(f, binary.BigEndian, &sb); err != nil {
        fmt.Println("Error al leer el superbloque:", err)
        return
    }

    if sb.S_filesystem_type != 3 {
        fmt.Println("Error: La partición no utiliza journaling (no es 3fs).")
        return
    }

    entrySize := int64(binary.Size(structs.JournalEntry{}))
    // journaling starts right after superblock (mkfs wrote it allí)
    journalStart := partitionStart + int64(binary.Size(sb))
    // number of entries: use inodes_count (mkfs looped n entries)
    totalEntries := int64(sb.S_inodes_count)

    // Move to journaling start
    if _, err := f.Seek(journalStart, 0); err != nil {
        fmt.Println("Error al posicionar el journaling:", err)
        return
    }

    // Build HTML table
    fmt.Println("<!doctype html>")
    fmt.Println("<html><head><meta charset=\"utf-8\"><title>Journaling</title></head><body>")
    fmt.Println("<h2>Journaling - Partición:", mountedPartition.Name, "</h2>")
    fmt.Println("<table border=\"1\" style=\"border-collapse:collapse;\">")
    fmt.Println("<thead><tr><th>Operacion</th><th>Path</th><th>Contenido</th><th>Fecha</th></tr></thead>")
    fmt.Println("<tbody>")

    for i := int64(0); i < totalEntries; i++ {
        var entry structs.JournalEntry
        if err := binary.Read(f, binary.BigEndian, &entry); err != nil {
            // si no se puede leer más, detener
            break
        }

        if entry.JCount == 0 {
            // entrada vacía
            continue
        }

        op := string(bytes.Trim(entry.JContent.IOperation[:], "\x00"))
        path := string(bytes.Trim(entry.JContent.IPath[:], "\x00"))
        content := string(bytes.Trim(entry.JContent.IContent[:], "\x00"))
        // si no es archivo, mostrar "-" para contenido
        if op == "" {
            op = "-"
        }
        if path == "" {
            path = "-"
        }
        if content == "" {
            content = "-"
        }

        // fecha: IDate se guarda como float (epoch). Convertir a readable.
        t := time.Unix(int64(entry.JContent.IDate), 0).Format("2006-01-02 15:04:05")

        // escapar mínimo (reemplazar '<' y '>' para evitar romper HTML)
        escape := func(s string) string {
			s = strings.ReplaceAll(s, "<", "&lt;")
			s = strings.ReplaceAll(s, ">", "&gt;")
			return s
		}

        fmt.Printf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
            escape(op), escape(path), escape(content), t)

        // avanzar al siguiente entry (binary.Read ya avanzó)
        _ = entrySize // mantenido por claridad (no necesario aquí)
    }

    fmt.Println("</tbody></table>")
    fmt.Println("</body></html>")
}
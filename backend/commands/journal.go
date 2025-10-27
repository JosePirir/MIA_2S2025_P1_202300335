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

// ShowJournal lee el journaling de la partición indicada y genera tabla HTML
func ShowJournal(id string) {
	mountedPartition, found := state.GetMountedPartitionByID(id)
	if !found {
		fmt.Printf("Error: No se encontró la partición montada con el id '%s'.\n", id)
		return
	}

	file, err := os.Open(mountedPartition.Path)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	// Leer superbloque
	var sb structs.Superblock
	if _, err := file.Seek(mountedPartition.Start, 0); err != nil {
		fmt.Println("Error al posicionar el disco:", err)
		return
	}
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	if sb.S_filesystem_type != 3 {
		fmt.Println("Error: la partición no es journaling (3fs).")
		return
	}

	entrySize := int64(binary.Size(structs.JournalEntry{}))
	journalStart := mountedPartition.Start + int64(binary.Size(sb))
	totalEntries := int64(sb.S_inodes_count)

	// HTML inicial
	fmt.Println("<!doctype html>")
	fmt.Println("<html><head><meta charset=\"utf-8\"><title>Journaling</title></head><body>")
	fmt.Printf("<h2>Journaling - Partición: %s</h2>\n", mountedPartition.Name)
	fmt.Println("<table border='1' style='border-collapse:collapse;'>")
	fmt.Println("<thead><tr><th>Operacion</th><th>Path</th><th>Contenido</th><th>Fecha</th></tr></thead>")
	fmt.Println("<tbody>")

	escape := func(s string) string {
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		return s
	}

	for i := int64(0); i < totalEntries; i++ {
		if _, err := file.Seek(journalStart+i*entrySize, 0); err != nil {
			break
		}

		var entry structs.JournalEntry
		if err := binary.Read(file, binary.BigEndian, &entry); err != nil {
			break
		}

		if entry.JCount == 0 {
			continue
		}

		// convertir campos a string
		op := strings.TrimRight(string(entry.JContent.IOperation[:]), "\x00")
		path := strings.TrimRight(string(entry.JContent.IPath[:]), "\x00")
		content := strings.TrimRight(string(entry.JContent.IContent[:]), "\x00")

		if op == "" {
			op = "-"
		}
		if path == "" {
			path = "-"
		}
		if content == "" {
			content = "-"
		}

		// convertir fecha
		t := time.Unix(int64(entry.JContent.IDate), 0).Format("2006-01-02 15:04:05")

		fmt.Printf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
			escape(op), escape(path), escape(content), t)
	}

	fmt.Println("</tbody></table></body></html>")
}


func addJournalEntry(file *os.File, sb structs.Superblock, sbStart int64, op string, src string, content string) {
    // Calculamos dónde empieza el journaling
    journalStart := sbStart + int64(binary.Size(sb))
    entrySize := int64(binary.Size(structs.JournalEntry{}))

    // Buscamos la primera entrada libre
    for i := int64(0); i < int64(sb.S_inodes_count); i++ {
        // Posicionamos en la entrada i
        file.Seek(journalStart+i*entrySize, 0)

        var entry structs.JournalEntry
        binary.Read(file, binary.BigEndian, &entry)

        if entry.JCount == 0 {
            // limpiar arrays
            for j := range entry.JContent.IOperation {
                entry.JContent.IOperation[j] = 0
            }
            for j := range entry.JContent.IPath {
                entry.JContent.IPath[j] = 0
            }
            for j := range entry.JContent.IContent {
                entry.JContent.IContent[j] = 0
            }

            // copiar valores
            copy(entry.JContent.IOperation[:], op)
            copy(entry.JContent.IPath[:], src)
            copy(entry.JContent.IContent[:], content)
            entry.JContent.IDate = float64(time.Now().Unix())
            entry.JCount = 1

            // escribir de vuelta
            file.Seek(journalStart+i*entrySize, 0)
            binary.Write(file, binary.BigEndian, &entry)
            break
        }
    }
}

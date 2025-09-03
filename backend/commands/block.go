package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
	"strings"
)

// escapeLabel escapa caracteres especiales para Graphviz
func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `<`, `\<`)
	s = strings.ReplaceAll(s, `>`, `\>`)
	s = strings.ReplaceAll(s, `{`, `\{`)
	s = strings.ReplaceAll(s, `}`, `\}`)
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// BLOCK genera un reporte gráfico de los bloques utilizados en la partición indicada
func BLOCK(id, path string) {
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

	// 2. Leer superbloque de la partición montada
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// 3. Leer bitmap de bloques
	bmBlocks := make([]byte, sb.S_blocks_count)
	file.Seek(int64(sb.S_bm_block_start), 0)
	if err := binary.Read(file, binary.BigEndian, &bmBlocks); err != nil {
		fmt.Println("Error al leer el bitmap de bloques:", err)
		return
	}

	// 4. Crear archivo DOT temporal
	dotFile := path + ".dot"
	f, err := os.Create(dotFile)
	if err != nil {
		fmt.Println("Error al crear el archivo DOT:", err)
		return
	}
	defer f.Close()

	fmt.Fprintln(f, "digraph BLOCKS {")
	fmt.Fprintln(f, "rankdir=LR;")
	fmt.Fprintln(f, "node [shape=record, fontsize=10, style=filled, fillcolor=lightblue];")

// 5. Generar nodos para cada bloque
var previous int = -1
for i := 0; i < int(sb.S_blocks_count); i++ {
    if bmBlocks[i] == 1 {
        var label string

        // Intentar leer como FolderBlock
        fb, errFolder := fs.ReadFolderBlock(file, sb, int32(i))
        if errFolder == nil {
            label = fmt.Sprintf("Bloque %d | {b_name | b_inodo\\l", i)
            for _, entry := range fb.B_content {
                name := strings.TrimRight(string(entry.B_name[:]), "\x00")
                label += fmt.Sprintf("%s | %d\\l", escapeLabel(name), entry.B_inodo)
            }
            label += "}"
        } else {
            // Intentar leer como FileBlock
            fblock, errFile := fs.ReadFileBlock(file, sb, int32(i))
            if errFile == nil {
                content := escapeLabel(string(fblock.B_content[:]))
                if len(content) > 50 {
                    content = content[:50] + "..."
                }
                label = fmt.Sprintf("Bloque %d | {Contenido\\l%s\\l}", i, content)
            } else {
                // Bloque de Apuntadores: imprimir bytes en tabla 4x4
                rawContent := make([]byte, 64)
                file.Seek(int64(sb.S_block_start)+int64(i*64), 0)
                file.Read(rawContent)
                label = fmt.Sprintf("Bloque %d | {", i)
                for j := 0; j < len(rawContent); j++ {
                    label += fmt.Sprintf("%d", rawContent[j])
                    if (j+1)%4 == 0 && j != len(rawContent)-1 {
                        label += "\\l"
                    } else if j != len(rawContent)-1 {
                        label += " | "
                    }
                }
                label += "}"
            }
        }

        // Escribir nodo en DOT
        fmt.Fprintf(f, "block%d [label=\"%s\"];\n", i, label)

        // Conectar con el bloque anterior
        if previous != -1 {
            fmt.Fprintf(f, "block%d -> block%d;\n", previous, i)
        }
        previous = i
    }
}

	fmt.Fprintln(f, "}")

	// 6. Generar imagen PNG
	imgFile := path + ".png"
	cmd := exec.Command("dot", "-Tpng", dotFile, "-o", imgFile)
	if err := cmd.Run(); err != nil {
		fmt.Println("Error al generar la imagen con Graphviz:", err)
		return
	}

	fmt.Println("Reporte de bloques generado en:", imgFile)
}

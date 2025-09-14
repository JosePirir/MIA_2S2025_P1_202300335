package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
	"strings"
)

// escapeDotLabel escapa caracteres especiales para Graphviz
func escapeDotLabel(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")   // quitar saltos de línea
	s = strings.ReplaceAll(s, "→", "->")   // normalizar flecha
	s = strings.ReplaceAll(s, "\\", "\\\\") // escapar backslash
	s = strings.ReplaceAll(s, "\"", "\\\"") // escapar comillas dobles
	return s
}

func TREE(id, path string) {
	// 1. Buscar partición montada
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

	// 2. Leer superbloque
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// 3. Leer bitmaps
	bmInodes := make([]byte, sb.S_inodes_count)
	file.Seek(int64(sb.S_bm_inode_start), 0)
	binary.Read(file, binary.BigEndian, &bmInodes)

	bmBlocks := make([]byte, sb.S_blocks_count)
	file.Seek(int64(sb.S_bm_block_start), 0)
	binary.Read(file, binary.BigEndian, &bmBlocks)

	// 4. Preparar salida
	outDir := filepath.Dir(path)
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		os.MkdirAll(outDir, 0755)
	}

	dotFile := path + ".dot"
	f, err := os.Create(dotFile)
	if err != nil {
		fmt.Println("Error al crear el archivo DOT:", err)
		return
	}
	defer f.Close()

	// 5. Encabezado del grafo
	fmt.Fprintln(f, "digraph G {")
	fmt.Fprintln(f, "    rankdir=TB;") // de arriba hacia abajo
	fmt.Fprintln(f, "    node [shape=record, fontsize=10, style=filled];")
	fmt.Fprintln(f, "    edge [arrowhead=vee, arrowsize=0.7];")
	fmt.Fprintf(f, "    labelloc=\"t\";\n")
	fmt.Fprintf(f, "    label=\"Reporte TREE - Partición %s (%s)\";\n", mountedPartition.Name, mountedPartition.ID)

	// Subgrafos
	fmt.Fprintln(f, "    subgraph cluster_inodes {")
	fmt.Fprintln(f, "        label=\"Inodos\";")
	fmt.Fprintln(f, "        color=blue; style=dashed;")

	// 6. Procesar inodos
	inodeSize := int64(binary.Size(structs.Inode{}))
	for i := 0; i < int(sb.S_inodes_count); i++ {
		if bmInodes[i] == 1 {
			var inode structs.Inode
			offset := int64(sb.S_inode_start) + int64(i)*inodeSize
			file.Seek(offset, 0)
			binary.Read(file, binary.BigEndian, &inode)

			label := fmt.Sprintf("Inodo %d | UID=%d | Size=%d | Type=%d | Perm=%d",
				i, inode.I_uid, inode.I_size, inode.I_type, inode.I_perm)

			fmt.Fprintf(f, "        inode%d [label=\"%s\", fillcolor=lightblue];\n",
				i, escapeDotLabel(label))

			// Relaciones inodo -> bloques
			for j, b := range inode.I_block {
				if b != -1 {
					fmt.Fprintf(f, "        inode%d -> block%d [label=\"%d\"];\n", i, b, j)
				}
			}
		}
	}
	fmt.Fprintln(f, "    }") // cierre cluster_inodes

	// Subgrafo de bloques
	fmt.Fprintln(f, "    subgraph cluster_blocks {")
	fmt.Fprintln(f, "        label=\"Bloques\";")
	fmt.Fprintln(f, "        color=red; style=dashed;")

	// 7. Procesar bloques
	for i := 0; i < int(sb.S_blocks_count); i++ {
		if bmBlocks[i] == 1 {
			var label string
			shape := "box"
			color := "lightgreen" // carpeta por defecto

			// Bloque carpeta
			if fb, err := fs.ReadFolderBlock(file, sb, int32(i)); err == nil {
				label = fmt.Sprintf("Bloque %d | Carpeta {", i)
				for _, entry := range fb.B_content {
					name := strings.TrimRight(string(entry.B_name[:]), "\x00")
					if name != "" && entry.B_inodo != -1 {
						label += fmt.Sprintf("%s -> inodo %d | ", escapeDotLabel(name), entry.B_inodo)
						fmt.Fprintf(f, "        block%d -> inode%d [label=\"%s\"];\n", i, entry.B_inodo, escapeDotLabel(name))
					}
				}
				label = strings.TrimRight(label, "| ")
				label += "}"

			} else if fblock, err := fs.ReadFileBlock(file, sb, int32(i)); err == nil {
				// Bloque archivo
				content := strings.TrimRight(string(fblock.B_content[:]), "\x00")
				if len(content) > 50 {
					content = content[:50] + "..."
				}
				label = fmt.Sprintf("Bloque %d | Archivo: %s", i, escapeDotLabel(content))
				color = "khaki"

			} else if apBlock, err := fs.ReadPointerBlock(file, sb, int32(i)); err == nil {
				// Bloque de apuntadores
				mainLabel := fmt.Sprintf("Bloque %d | Apuntadores", i)
				fmt.Fprintf(f, "        block%d [label=\"%s\", shape=box, fillcolor=lightcoral];\n",
					i, escapeDotLabel(mainLabel))

				for j, ptr := range apBlock {
					if ptr != -1 {
						childName := fmt.Sprintf("block%d_ptr%d", i, j)
						childLabel := fmt.Sprintf("Ptr %d -> %d", j, ptr)

						fmt.Fprintf(f, "        %s [label=\"%s\", shape=box, fillcolor=lightcoral];\n",
							childName, escapeDotLabel(childLabel))

						fmt.Fprintf(f, "        block%d -> %s;\n", i, childName)
						fmt.Fprintf(f, "        %s -> block%d;\n", childName, ptr)
					}
				}
				continue
			} else {
				label = fmt.Sprintf("Bloque %d | Desconocido", i)
				color = "white"
			}

			// Nodo del bloque
			fmt.Fprintf(f, "        block%d [label=\"%s\", shape=%s, fillcolor=%s];\n",
				i, escapeDotLabel(label), shape, color)
		}
	}
	fmt.Fprintln(f, "    }") // cierre cluster_blocks

	// Cierre grafo
	fmt.Fprintln(f, "}")

	// 8. Generar PNG
	imgFile := path + ".png"
	cmd := exec.Command("dot", "-Tpng", dotFile, "-o", imgFile)
	if err := cmd.Run(); err != nil {
		fmt.Println("Error al generar la imagen con Graphviz:", err)
		return
	}

	fmt.Println("Reporte TREE generado en:", imgFile)
}

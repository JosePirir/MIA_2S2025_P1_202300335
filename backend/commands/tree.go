// commands/tree.go
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

// TREE genera el árbol gráfico del sistema de archivos EXT2 de una partición montada
func TREE(id, path string) error {
	// 1. Buscar la partición montada
	mountedPartition, found := state.GetMountedPartitionByID(id)
	if !found {
		return fmt.Errorf("no se encontró partición montada con id %s", id)
	}

	// 2. Abrir archivo de disco
	file, err := os.Open(mountedPartition.Path)
	if err != nil {
		return fmt.Errorf("error abriendo el disco: %v", err)
	}
	defer file.Close()

	// 3. Leer Superblock
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.LittleEndian, &sb); err != nil {
		return fmt.Errorf("error leyendo superblock: %v", err)
	}

	// 4. Crear contenido DOT
	dot := "digraph G {\n"
	dot += "rankdir=TB;\n"
	dot += "node [shape=record, style=filled, fontname=Helvetica];\n"

	// 5. Iniciar desde el inodo raíz (#0 en EXT2)
	traverseInodeTree(file, sb, 0, &dot)

	dot += "}\n"

	// 6. Guardar DOT temporal y generar imagen con Graphviz
	tmpPath := "tree_report.dot"
	if err := os.WriteFile(tmpPath, []byte(dot), 0644); err != nil {
		return fmt.Errorf("error escribiendo archivo DOT: %v", err)
	}

	cmd := exec.Command("dot", "-Tpng", tmpPath, "-o", path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error ejecutando Graphviz: %v", err)
	}

	fmt.Printf("Reporte TREE generado en %s\n", path)
	return nil
}

// traverseInodeTree recorre recursivamente los inodos y sus bloques
func traverseInodeTree(file *os.File, sb structs.Superblock, inodeIndex int32, dot *string) {
	inode, err := fs.ReadInode(file, sb, inodeIndex)
	if err != nil {
		return
	}

	nodeName := fmt.Sprintf("inode%d", inodeIndex)
	color := "#FFCC00" // amarillo para inodos
	*dot += fmt.Sprintf("%s [label=\"{INODO %d|Tamaño: %d|Bloques: %d}\" fillcolor=\"%s\"];\n",
		nodeName, inodeIndex, inode.I_size, inode.I_block, color)

	// recorrer bloques asociados
	for _, b := range inode.I_block {
		if b == -1 {
			continue
		}

		blockName := fmt.Sprintf("block%d", b)

		// Detectar si es carpeta o archivo
		if inode.I_type == 1 { // Carpeta
			*dot += fmt.Sprintf("%s [label=\"Bloque Carpeta %d\" fillcolor=\"#99CCFF\"];\n", blockName, b)
		} else {
			*dot += fmt.Sprintf("%s [label=\"Bloque Archivo %d\" fillcolor=\"#CCFF99\"];\n", blockName, b)
		}

		*dot += fmt.Sprintf("%s -> %s;\n", nodeName, blockName)

		// Si es carpeta, leer FolderBlock y seguir recorriendo
		if inode.I_type == 1 {
			fb, err := fs.ReadFolderBlock(file, sb, b)
			if err != nil {
				continue
			}
			for _, content := range fb.B_content {
				if content.B_inodo != -1 {
					name := strings.TrimSpace(string(content.B_name[:])) // convertir [12]byte → string
					if name != "" {
						childName := fmt.Sprintf("inode%d", content.B_inodo)
						*dot += fmt.Sprintf("%s -> %s [label=\"%s\"];\n", blockName, childName, name)
						traverseInodeTree(file, sb, content.B_inodo, dot)
					}
				}
			}
		}
	}
}

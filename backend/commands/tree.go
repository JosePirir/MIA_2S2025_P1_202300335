// commands/tree.go
package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"proyecto1/state"
	"proyecto1/structs"
	"proyecto1/fs"
)

// TREE genera un reporte gráfico de los inodos y carpetas (árbol del sistema de archivos)
func TREE(id, path string) error {
	// 1. Buscar la partición montada
	mountedPartition, found := state.GetMountedPartitionByID(id)
	if !found {
		return fmt.Errorf("no se encontró la partición montada con ID %s", id)
	}

	// 2. Abrir el disco
	file, err := os.Open(mountedPartition.Path)
	if err != nil {
		return fmt.Errorf("error abriendo el disco: %v", err)
	}
	defer file.Close()

	// 3. Leer el superbloque
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	err = binary.Read(file, binary.LittleEndian, &sb)
	if err != nil {
		return fmt.Errorf("error leyendo superbloque: %v", err)
	}

	// 4. Iniciar DOT
	dot := "digraph TREE {\n"
	dot += "node [shape=plaintext fontname=\"Helvetica\"];\n"

	// 5. Llamar a recursivo desde inodo raíz
	dot += recorrerInodo(file, sb, 0)

	dot += "}\n"

	// 6. Guardar en archivo DOT
	dotFile := "/tmp/tree.dot"
	imgFile := path
	err = os.WriteFile(dotFile, []byte(dot), 0644)
	if err != nil {
		return fmt.Errorf("error escribiendo archivo DOT: %v", err)
	}

	// 7. Generar imagen con Graphviz
	cmd := exec.Command("dot", "-Tpng", dotFile, "-o", imgFile)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error ejecutando Graphviz: %v", err)
	}

	return nil
}

// recorrerInodo dibuja un inodo y sus bloques/carpetas hijos
func recorrerInodo(file *os.File, sb structs.Superblock, index int32) string {
	dot := ""

	// 1. Leer inodo
	inode, err := fs.ReadInode(file, sb, index)
	if err != nil {
		return ""
	}

	// 2. Crear tabla del inodo
	inodeName := fmt.Sprintf("inode%d", index)
	dot += fmt.Sprintf("%s [label=<\n", inodeName)
	dot += "<table border='1' cellborder='1' cellspacing='0'>\n"
	dot += fmt.Sprintf("<tr><td colspan='2'>Inodo %d</td></tr>\n", index)

	for i, ptr := range inode.I_block {
		dot += fmt.Sprintf("<tr><td>AP%d</td><td>%d</td></tr>\n", i, ptr)
	}
	dot += "</table>>];\n"

	// 3. Recorrer punteros
	for i, ptr := range inode.I_block {
		fmt.Printf("Puntero %d: %d\n", i, ptr)
		if ptr != -1 {
			// Leer bloque carpeta
			block, err := fs.ReadFolderBlock(file, sb, ptr)
			if err != nil {
				continue
			}

			// Dibujar bloque
			blockName := fmt.Sprintf("block%d", ptr)
			dot += fmt.Sprintf("%s [label=<\n", blockName)
			dot += "<table border='1' cellborder='1' cellspacing='0'>\n"
			dot += fmt.Sprintf("<tr><td colspan='2'>Bloque %d</td></tr>\n", ptr)

			for _, content := range block.B_content {
				dot += fmt.Sprintf("<tr><td>%s</td><td>%d</td></tr>\n", content.B_name, content.B_inodo)
			}
			dot += "</table>>];\n"

			// Relación inodo -> bloque
			dot += fmt.Sprintf("%s -> %s;\n", inodeName, blockName)

			// Recursión a inodos hijos
			for _, content := range block.B_content {
				if content.B_inodo != -1 {
					dot += recorrerInodo(file, sb, content.B_inodo)
					dot += fmt.Sprintf("%s -> inode%d;\n", blockName, content.B_inodo)
				}
			}
		}
	}

	return dot
}

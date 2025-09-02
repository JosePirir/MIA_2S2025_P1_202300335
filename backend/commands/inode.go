package commands

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"proyecto1/state"
	"proyecto1/structs"
)

func INODE(id, path string) {
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

	// 2. Leer el superbloque
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// 3. Leer el bitmap de inodos
	bmInodes := make([]byte, sb.S_inodes_count)
	file.Seek(int64(sb.S_bm_inode_start), 0)
	if err := binary.Read(file, binary.BigEndian, &bmInodes); err != nil {
		fmt.Println("Error al leer el bitmap de inodos:", err)
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

	fmt.Fprintln(f, "digraph INODES {")
	fmt.Fprintln(f, "rankdir=LR;")
	fmt.Fprintln(f, "node [shape=record, fontsize=10];")

	// 5. Recorrer inodos
	inodeSize := int64(binary.Size(structs.Inode{}))
	previous := -1
	for i := 0; i < int(sb.S_inodes_count); i++ {
		if bmInodes[i] == 1 {
			var inode structs.Inode
			offset := int64(sb.S_inode_start) + int64(i)*inodeSize
			file.Seek(offset, 0)
			if err := binary.Read(file, binary.BigEndian, &inode); err != nil {
				fmt.Printf("Error al leer el inodo %d: %v\n", i, err)
				continue
			}

			// Construir label sin los bloques
			label := fmt.Sprintf(
				"Inodo %d | UID=%d | GID=%d | Size=%d | Atime=%d | Ctime=%d | Mtime=%d | Type=%d | Perm=%d",
				i, inode.I_uid, inode.I_gid, inode.I_size,
				inode.I_atime, inode.I_ctime, inode.I_mtime,
				inode.I_type, inode.I_perm,
			)

			fmt.Fprintf(f, "inode%d [label=\"%s\"];\n", i, label)

			// Conectar con el anterior inodo
			if previous != -1 {
				fmt.Fprintf(f, "inode%d -> inode%d;\n", previous, i)
			}
			previous = i
		}
	}

	fmt.Fprintln(f, "}")

	// 6. Generar imagen PNG directamente
	imgFile := path + ".png"
	cmd := exec.Command("dot", "-Tpng", dotFile, "-o", imgFile)
	if err := cmd.Run(); err != nil {
		fmt.Println("Error al generar la imagen con Graphviz:", err)
		return
	}

	fmt.Println("Reporte de inodos generado en:", imgFile)
}

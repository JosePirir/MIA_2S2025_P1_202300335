package fs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"proyecto1/structs"
	"os"
	"strings"
)

// FindInodeByPath navega el sistema de archivos para encontrar el inodo de una ruta específica.
func FindInodeByPath(file *os.File, sb structs.Superblock, path string) (structs.Inode, int32, error) {
	if !strings.HasPrefix(path, "/") {
		return structs.Inode{}, -1, errors.New("la ruta debe ser absoluta (empezar con /)")
	}

	currentInodeIndex := int32(0) // Empezamos desde el inodo raíz (0)
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if path == "/" {
		pathParts = []string{}
	}

	for _, part := range pathParts {
		if part == "" {
			continue
		}
		inode, err := ReadInode(file, sb, currentInodeIndex)
		if err != nil {
			return structs.Inode{}, -1, err
		}
		if inode.I_type != 0 { // 0 es para carpeta
			return structs.Inode{}, -1, errors.New("la ruta contiene un archivo en una posición intermedia")
		}

		foundNext := false
		for _, blockPtr := range inode.I_block {
			if blockPtr == -1 {
				continue
			}
			folderBlock, err := ReadFolderBlock(file, sb, blockPtr)
			if err != nil {
				return structs.Inode{}, -1, err
			}
			for _, entry := range folderBlock.B_content {
				if entry.B_inodo != -1 && strings.TrimRight(string(entry.B_name[:]), "\x00") == part {
					currentInodeIndex = entry.B_inodo
					foundNext = true
					break
				}
			}
			if foundNext {
				break
			}
		}
		if !foundNext {
			return structs.Inode{}, -1, errors.New("no se encontró el archivo o directorio: " + part)
		}
	}

	finalInode, err := ReadInode(file, sb, currentInodeIndex)
	return finalInode, currentInodeIndex, err
}

// ReadFileContent lee todos los bloques de datos de un inodo y devuelve su contenido.
func ReadFileContent(file *os.File, sb structs.Superblock, inode structs.Inode) ([]byte, error) {
	if inode.I_type != 1 { // 1 es para archivo
		return nil, errors.New("el inodo no corresponde a un archivo")
	}
	var content bytes.Buffer
	// solo leemos punteros directos
	for i := 0; i < 12 && inode.I_block[i] != -1; i++ {
		blockPtr := inode.I_block[i]
		fileBlock, err := ReadFileBlock(file, sb, blockPtr)
		if err != nil {
			return nil, err
		}
		content.Write(fileBlock.B_content[:])
	}
	// Devolver solo la cantidad de bytes especificada por el tamaño del inodo
	if int64(content.Len()) > int64(inode.I_size) {
		return content.Bytes()[:inode.I_size], nil
	}
	return content.Bytes(), nil
}

// --- Funciones auxiliares de lectura de bajo nivel ---
func ReadInode(file *os.File, sb structs.Superblock, index int32) (structs.Inode, error) {
	var inode structs.Inode
	offset := int64(sb.S_inode_start) + int64(index)*int64(sb.S_inode_size)
	file.Seek(offset, 0)
	err := binary.Read(file, binary.BigEndian, &inode)
	return inode, err
}
func ReadFileBlock(file *os.File, sb structs.Superblock, index int32) (structs.FileBlock, error) {
	var block structs.FileBlock
	offset := int64(sb.S_block_start) + int64(index)*int64(sb.S_block_size)
	file.Seek(offset, 0)
	err := binary.Read(file, binary.BigEndian, &block)
	return block, err
}
func ReadFolderBlock(file *os.File, sb structs.Superblock, index int32) (structs.FolderBlock, error) {
	var block structs.FolderBlock
	offset := int64(sb.S_block_start) + int64(index)*int64(sb.S_block_size)
	file.Seek(offset, 0)
	err := binary.Read(file, binary.BigEndian, &block)
	return block, err
}
// WriteInode guarda un inodo en disco en una posición específica
func WriteInode(file *os.File, sb structs.Superblock, index int32, inode structs.Inode) error {
    offset := int64(sb.S_inode_start) + int64(index)*int64(sb.S_inode_size)
    file.Seek(offset, 0)
    return binary.Write(file, binary.BigEndian, &inode)
}

// WriteFileBlock guarda un bloque de archivo en disco
func WriteFileBlock(file *os.File, sb structs.Superblock, index int32, block structs.FileBlock) error {
    offset := int64(sb.S_block_start) + int64(index)*int64(sb.S_block_size)
    file.Seek(offset, 0)
    return binary.Write(file, binary.BigEndian, &block)
}
// FindFreeBlock busca el primer bloque libre en el bitmap
// Busca primer bloque libre (byte 0)
func FindFreeBlock(file *os.File, sb structs.Superblock) (int32, error) {
    bitmap := make([]byte, sb.S_blocks_count)
    if _, err := file.ReadAt(bitmap, int64(sb.S_bm_block_start)); err != nil {
        return -1, err
    }
    for i := int32(0); i < sb.S_blocks_count; i++ {
        if bitmap[i] == 0 { // 0 = libre
            return i, nil
        }
    }
    return -1, errors.New("no hay bloques libres disponibles")
}

// MarkBlockAsUsed marca un bloque en el bitmap como ocupado
func MarkBlockAsUsed(file *os.File, sb structs.Superblock, index int32) error {
    _, err := file.WriteAt([]byte{1}, int64(sb.S_bm_block_start)+int64(index))
    return err
}

func MarkBlockAsFree(file *os.File, sb structs.Superblock, index int32) error {
    _, err := file.WriteAt([]byte{0}, int64(sb.S_bm_block_start)+int64(index))
    return err
}

func FindFreeInode(file *os.File, sb structs.Superblock) (int32, error) {
    bitmap := make([]byte, sb.S_inodes_count)
    if _, err := file.ReadAt(bitmap, int64(sb.S_bm_inode_start)); err != nil {
        return -1, err
    }
    for i := int32(0); i < sb.S_inodes_count; i++ {
        if bitmap[i] == 0 {
            return i, nil
        }
    }
    return -1, errors.New("no hay inodos libres disponibles")
}

func MarkInodeAsUsed(file *os.File, sb structs.Superblock, index int32) error {
    _, err := file.WriteAt([]byte{1}, int64(sb.S_bm_inode_start)+int64(index))
    return err
}

func WriteFolderBlock(file *os.File, sb structs.Superblock, blockIndex int32, fb structs.FolderBlock) error {
    blockSize := sb.S_block_size
    offset := int64(sb.S_block_start) + int64(blockIndex)*int64(blockSize)
    file.Seek(offset, 0)
    return binary.Write(file, binary.BigEndian, &fb)
}

func ReadPointerBlock(file *os.File, sb structs.Superblock, index int32) ([]int32, error) {
    blockSize := int(sb.S_block_size)
    offset := int64(sb.S_block_start) + int64(index)*int64(blockSize)

    buf := make([]byte, blockSize)
    if _, err := file.ReadAt(buf, offset); err != nil {
        return nil, err
    }

    // cada apuntador ocupa 4 bytes (int32)
    count := blockSize / 4
    pointers := make([]int32, 0, count)
    for i := 0; i < count; i++ {
        b := buf[i*4 : (i+1)*4]
        // convertimos a uint32 y luego a int32 para conservar el valor negativo (ej. -1 -> 0xFFFFFFFF)
        v := int32(binary.BigEndian.Uint32(b))
        pointers = append(pointers, v)
    }
    return pointers, nil
}
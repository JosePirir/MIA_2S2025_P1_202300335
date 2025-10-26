package structs

const NAME_MAX = 64 // ajustar según lo que necesites

// Asegura que el FileBlock sea mayor o igual al tamaño del FolderBlock.
// Cada ContentEntry ocupa NAME_MAX bytes + 4 bytes para B_inodo.
// FolderBlock tiene 4 entradas, así que reservar suficiente espacio para FileBlock.
const FILE_BLOCK_SIZE = 4*NAME_MAX + 16

type ContentEntry struct {
	B_name  [NAME_MAX]byte // Nombre del archivo o carpeta (ampliado)
	B_inodo int32          // Apuntador al inodo
}

// FolderBlock es la estructura para un bloque de directorio.
type FolderBlock struct {
	B_content [4]ContentEntry
}

// FileBlock es la estructura para un bloque de contenido de archivo.
// Aumentado para que su tamaño >= FolderBlock y evitar truncamientos.
type FileBlock struct {
	B_content [FILE_BLOCK_SIZE]byte
}

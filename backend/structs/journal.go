package structs

// JournalEntry representa una entrada en el journaling del sistema de archivos EXT3.
type JournalEntry struct {
    JCount   int32       // <- tamaño fijo (4 bytes)
    JContent Information // Contiene toda la información de la acción realizada.
}

// Information contiene los detalles de una operación registrada en el journaling.
type Information struct {
    IOperation [10]byte  // Operación realizada (e.g., "create", "delete").
    IPath      [32]byte  // Ruta donde se realizó la operación.
    IContent   [64]byte  // Contenido asociado (si aplica, como el contenido de un archivo).
    IDate      float64   // Fecha en la que se realizó la operación.
}
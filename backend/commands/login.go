package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"proyecto1/fs"
	"proyecto1/state"
	"proyecto1/structs"
	"strings"
)

func ExecuteLogin(user, pass, id string) {
	if state.CurrentSession.IsActive {
		fmt.Println("Error: Ya hay una sesión iniciada. Cierra sesión antes de iniciar otra.")
		return
	}

	if len(state.GlobalMountedPartitions) == 0 {
		fmt.Println("No hay particiones montadas.")
		return
	}

	var mountedPartition *state.MountedPartition
	for _, p := range state.GlobalMountedPartitions {
		if p.ID == id {
			mountedPartition = &p
			break
		}
	}
	if mountedPartition == nil {
		fmt.Printf("No existe la particion con el id %s\n", id)
		return
	}
	fmt.Printf("Particion encontrada %s en %s\n", id, mountedPartition.Path)

	file, err := os.OpenFile(mountedPartition.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	// Leer superbloque
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// === Leer TODO el contenido de /users.txt desde su inodo ===
	usersContent, err := readUsersTxt(file, sb)
	if err != nil {
		fmt.Println("Error al leer users.txt:", err)
		return
	}

	// === Parseo robusto (quitando comillas y espacios) ===
	lines := strings.Split(usersContent, "\n")
	loginSuccess := false

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		// Esperado: id,tipo,grupo,usuario,pass
		// Usamos SplitN para no rebanar de más si hubiera comas internas (5 campos máximo)
		fields := strings.SplitN(line, ",", 5)
		if len(fields) < 5 {
			continue
		}

		tipo := strings.TrimSpace(fields[1])
		if tipo != "U" {
			continue
		}

		username := trimQuotesSpaces(fields[3])
		password := trimQuotesSpaces(fields[4])

		if username == user && password == pass {
			loginSuccess = true
			break
		}
	}

	if loginSuccess {
		state.CurrentSession.User = user
		state.CurrentSession.PartitionID = id
		state.CurrentSession.IsActive = true
		fmt.Printf("Login exitoso para el usuario '%s'\n", user)
	} else {
		fmt.Println("Usuario o contraseña incorrectos.")
	}
}

// Busca el inodo de /users.txt y concatena todos los bloques de archivo.
func readUsersTxt(file *os.File, sb structs.Superblock) (string, error) {
	inode, _, err := fs.FindInodeByPath(file, sb, "/users.txt")
	if err != nil {
		return "", fmt.Errorf("no se encontró /users.txt: %w", err)
	}

	var buf bytes.Buffer
	for _, bptr := range inode.I_block {
		if bptr == -1 {
			continue
		}
		fb, err := fs.ReadFileBlock(file, sb, bptr)
		if err != nil {
			return "", fmt.Errorf("error al leer bloque %d de users.txt: %w", bptr, err)
		}
		// Quita bytes nulos de cada bloque antes de añadirlos.
		buf.Write(bytes.TrimRight(fb.B_content[:], "\x00"))
	}

	// Limpieza final (por si quedaron nulos al final)
	return string(bytes.TrimRight(buf.Bytes(), "\x00")), nil
}

// Recorta espacios y comillas envolventes (ej. "usuario1" -> usuario1).
func trimQuotesSpaces(s string) string {
	return strings.Trim(s, " \t\r\n\"")
}

func ExecuteLogout() {
	if !state.CurrentSession.IsActive {
		fmt.Println("Error: No hay ninguna sesión activa.")
		return
	}

	fmt.Printf("Cerrando sesión del usuario '%s' en la partición '%s'\n",
		state.CurrentSession.User, state.CurrentSession.PartitionID)

	// Limpiamos la sesión
	state.CurrentSession.User = ""
	state.CurrentSession.PartitionID = ""
	state.CurrentSession.IsActive = false
}

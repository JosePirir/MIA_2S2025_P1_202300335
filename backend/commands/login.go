package commands

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"proyecto1/state"
	"proyecto1/structs"
	"strings"
	//"time"
)

func ExecuteLogin(user string, pass string, id string) {

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

	// Abrimos el disco
	file, err := os.OpenFile(mountedPartition.Path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco:", err)
		return
	}
	defer file.Close()

	// Leemos el superbloque
	var sb structs.Superblock
	file.Seek(mountedPartition.Start, 0)
	if err := binary.Read(file, binary.BigEndian, &sb); err != nil {
		fmt.Println("Error al leer el superbloque:", err)
		return
	}

	// Leemos el bloque de users.txt (Bloque 1)
	blockPos := sb.S_block_start + int32(1*sb.S_block_size)
	usersBlock := make([]byte, sb.S_block_size)
	file.Seek(int64(blockPos), 0)
	if _, err := io.ReadFull(file, usersBlock); err != nil {
		fmt.Println("Error al leer users.txt:", err)
		return
	}

	// Convertimos a string y separamos por líneas
	usersContent := string(bytes.Trim(usersBlock, "\x00")) // eliminar bytes nulos
	lines := strings.Split(usersContent, "\n")

	// Buscamos usuario
	loginSuccess := false
	for _, line := range lines {
		fields := strings.Split(line, ",")
		if len(fields) >= 5 && fields[1] == "U" { // Solo usuarios
			username := strings.TrimSpace(fields[3])
			password := strings.TrimSpace(fields[4])
			if username == user && password == pass {
				loginSuccess = true
				break
			}
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

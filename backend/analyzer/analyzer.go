package analyzer

import (
	"bufio"
	"flag"
	"fmt"
	"proyecto1/commands"
	"io"
	"os"
	"strings"
)

// ProcessCommands recibe un string con comandos y los procesa línea por línea
// Devuelve la salida completa como string
func ProcessCommands(input string) string {
	var outputBuilder strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(input))

	for scanner.Scan() {
		line := scanner.Text()

		// Ignora líneas vacías
		if strings.TrimSpace(line) == "" {
			
		}

		// Si es un comentario, ignorarlo
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			
		}

		// Procesa la línea de comando actual
		outputBuilder.WriteString(fmt.Sprintf("> %s\n", line))

		// Si el usuario quiere salir, retornamos inmediatamente
		if strings.ToLower(line) == "exit" {
			outputBuilder.WriteString("Saliendo...\n")
			
		}

		// Ejecuta el comando y captura su salida
		output := executeCommand(line)
		outputBuilder.WriteString(output)
		outputBuilder.WriteString("\n")
	}

	return outputBuilder.String()
}

func executeCommand(commandLine string) string {
	// Divide la línea en partes (comando y argumentos)
	parts := strings.Fields(commandLine)
	if len(parts) == 0 {
		return ""
	}

	// La primera palabra es el comando
	command := strings.ToLower(parts[0])
	args := parts[1:]

	// Guarda stdout original para restaurarlo después
	oldStdout := os.Stdout

	// Crea un pipe para capturar stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
		switch command {
		case "mkdisk":
			// Crear un nuevo conjunto de flags específico para el comando mkdisk
			// flag.ContinueOnError hace que el programa termine si hay un error en los flags
			mkdiskCmd := flag.NewFlagSet("mkdisk", flag.ContinueOnError)

			// Definir los flags que acepta el comando mkdisk:
			// Int() crea un flag que acepta números enteros
			size := mkdiskCmd.Int("size", 0, "Tamaño del disco.")

			// String() crea un flag que acepta cadenas de texto
			unit := mkdiskCmd.String("unit", "m", "Unidad del tamaño (k/m).")
			fit := mkdiskCmd.String("fit", "ff", "Tipo de ajuste (bf/ff/wf).")
			path := mkdiskCmd.String("path", "", "Ruta del disco a crear.")

			// Parse() analiza los argumentos y llena las variables con los valores correspondientes
			mkdiskCmd.Parse(args)

			// El parámetro path es obligatorio, no puede estar vacío
			// *path desreferencia el puntero para obtener el valor real
			if *path == "" {
				fmt.Println("Error: el parámetro -path es obligatorio para mkdisk.")
				 // Vuelve al inicio del bucle sin ejecutar el comando
			}

			// El parámetro size debe ser positivo y mayor que cero
			if *size <= 0 {
				fmt.Println("Error: el parámetro -size es obligatorio y debe ser positivo.")
				 // Vuelve al inicio del bucle sin ejecutar el comando
			}

			// Si todas las validaciones pasan, ejecuta el comando mkdisk
			// Pasa los valores desreferenciados (con *) a la función
			commands.ExecuteMkdisk(*size, *unit, *fit, *path)

		case "rmdisk":
			// Crea un FlagSet específico para rmdisk.
			rmdiskCmd := flag.NewFlagSet("rmdisk", flag.ContinueOnError)
			// Define el único parámetro que rmdisk necesita: -path.
			path := rmdiskCmd.String("path", "", "Ruta del disco a eliminar.")

			// Parsea los argumentos después del comando "rmdisk".
			rmdiskCmd.Parse(args)

			// Valida que el parámetro -path se haya proporcionado.
			if *path == "" {
				fmt.Println("Error: el parámetro -path es obligatorio para rmdisk.")
				
			}
			// Ejecuta la lógica del comando rmdisk.
			commands.ExecuteRmdisk(*path)
		case "fdisk":
			// Crea un FlagSet para fdisk con todos sus parámetros.
			fdiskCmd := flag.NewFlagSet("fdisk", flag.ContinueOnError)
			size := fdiskCmd.Int64("size", 0, "Tamaño de la partición.")
			path := fdiskCmd.String("path", "", "Ruta del disco.")
			name := fdiskCmd.String("name", "", "Nombre de la partición.")
			unit := fdiskCmd.String("unit", "k", "Unidad del tamaño (b/k/m).")
			typeStr := fdiskCmd.String("type", "p", "Tipo de partición (p/e/l).")
			fit := fdiskCmd.String("fit", "wf", "Tipo de ajuste (bf/ff/wf).")
			delete := fdiskCmd.String("delete", "", "Tipo de delete (fast/full).")
			add := fdiskCmd.Int64("add", 0, "Tamaño agregar o quitar de una particion.")

			fdiskCmd.Parse(args)

			// Validar parámetros obligatorios
			if *path == "" || *name == "" {
				fmt.Println("Error: los parámetros -path, -name y -size son obligatorios para fdisk.")				
			}

			commands.ExecuteFdisk(*path, *name, *unit, *typeStr, *fit, *size, *delete, *add)
		// --- FIN DE LA MODIFICACIÓN ---
		case "mount":
			mountCmd := flag.NewFlagSet("mount", flag.ContinueOnError)
			path := mountCmd.String("path", "", "Ruta del disco.")
			name := mountCmd.String("name", "", "Nombre de la partición.")
			mountCmd.Parse(args)

			if *path == "" || *name == "" {
				fmt.Println("Error: los parámetros -path y -name son obligatorios.")
				
			}
			commands.ExecuteMount(*path, *name)

		case "unmount":
			unmountCmd := flag.NewFlagSet("unmounted", flag.ContinueOnError)
			id := unmountCmd.String("id", "", "ID de la particion.")
			unmountCmd.Parse(args)

			if *id == "" {
				fmt.Println("Error: el parametro -id es obligatorio.")
			}
			commands.ExecuteUnmount(*id)

		case "mounted":
			commands.ExecuteMounted()

		case "mkfs":
			// Crea un FlagSet específico para mkfs.
			mkfsCmd := flag.NewFlagSet("mkfs", flag.ContinueOnError)
			id := mkfsCmd.String("id", "", "ID de la partición a formatear.")
			typeStr := mkfsCmd.String("type", "full", "Tipo de formateo (full).")
			fs := mkfsCmd.String("fs", "2fs", "Sistema de archivos (2fs).")

			// Parsea los argumentos después del comando "mkfs".
			mkfsCmd.Parse(args)

			// Valida que el parámetro -id se haya proporcionado.
			if *id == "" {
				fmt.Println("Error: el parámetro -id es obligatorio para mkfs.")
				
			}
			// Ejecuta la lógica del comando mkfs.
			commands.ExecuteMkfs(*id, *typeStr, *fs)

		case "login":
			loginCmd := flag.NewFlagSet("login", flag.ContinueOnError)
			user := loginCmd.String("user", "", "Usuario que va a iniciar sesion.")
			pass := loginCmd.String("pass", "", "Contrasenia para iniciar sesion.")
			id := loginCmd.String("id", "", "ID de la particion en la que se va a iniciar sesion.")

			loginCmd.Parse(args)

			if *user == "" || *pass == "" || *id == "" {
				fmt.Println("Error: Los parametros -user, -pass y -id son obligatorios para login.")
				
			}

			commands.ExecuteLogin(*user, *pass, *id)

		case "logout":
			commands.ExecuteLogout()

		case "cat":
			catCmd := flag.NewFlagSet("cat", flag.ContinueOnError)
			file := catCmd.String("file", "", "File que se va a leer de la particion en la que previamente ya se inicio sesion.")

			catCmd.Parse(args)

			if *file == "" {
				fmt.Println("Error: El parametro -file es obligatorio para cat.")
				
			}

			commands.ExecuteCat(*file)

		case "mkgrp":
			mkgrpCmd := flag.NewFlagSet("mkgrp", flag.ContinueOnError)
			name := mkgrpCmd.String("name", "", "Nombre del grupo a crear en users.txt.")

			mkgrpCmd.Parse(args)

			if *name == "" {
				fmt.Println("Error: El parametro -name es obligatorio para mkgrp.")
				
			}

			commands.ExecuteMkgrp(*name)

		case "rmgrp":
			rmgrpCmd := flag.NewFlagSet("rmgrp", flag.ContinueOnError)
			name := rmgrpCmd.String("name", "", "Nombre del grupo a crear en users.txt.")

			rmgrpCmd.Parse(args)

			if *name == "" {
				fmt.Println("Error: El parametro -name es obligatorio para rmgrp.")
				
			}

			commands.ExecuteRmgrp(*name)

		case "mkusr":
			mkusrCmd := flag.NewFlagSet("mkusr", flag.ContinueOnError)
			user := mkusrCmd.String("user", "", "Nombre del usuario a crear.")
			pass := mkusrCmd.String("pass", "", "Contrasenia del usuario a crear.")
			grp := mkusrCmd.String("grp", "", "Grupo que sera el usuario.")

			mkusrCmd.Parse(args)

			if *user == "" || *pass == "" || *grp == "" {
				fmt.Println("Error: Los parametros -user, -pass y -grp son obligatorios para mkusr.")
				
			}

			commands.ExecuteMkusr(*user, *pass, *grp)

		case "rmusr":
			rmuserCmd := flag.NewFlagSet("rmusr", flag.ContinueOnError)
			user := rmuserCmd.String("user", "", "Nombre del usuario a eliminar.")

			rmuserCmd.Parse(args)

			if *user=="" {
				fmt.Println("Error: El parametro -user es obligatorio para rmusr.")
				
			}

			commands.ExecuteRmusr(*user)

		case "chgrp":
			chgrpCmd := flag.NewFlagSet("chgrp", flag.ContinueOnError)
			user := chgrpCmd.String("user", "", "Nombre del usuario a cambiar de grupo.")
			grp := chgrpCmd.String("grp", "", "Grupo al que se cambiara el usuario.")

			chgrpCmd.Parse(args)

			if *user=="" || *grp=="" {
				fmt.Println("Error: El parametro user y grp son obligatorios para chgrp.")
				
			}

			commands.ExecuteChgrp(*user, *grp)
		case "mkdir":
			mkdirCmd := flag.NewFlagSet("mkdir", flag.ContinueOnError)
			path := mkdirCmd.String("path","","Ruta de la carpeta que se creara.")
			p := mkdirCmd.Bool("p", false, "Si existe, se pueden crear directorios padres.")

			mkdirCmd.Parse(args)

			if *path=="" {
				fmt.Println("Error: El parametro path debe ser obligatorio para mkdir.")
				
			}
			
			commands.ExecuteMkdir(*path, *p)
			
		case "mkfile":
			mkfileCmd := flag.NewFlagSet("mkfile", flag.ContinueOnError)
			path := mkfileCmd.String("path", "", "Ruta donde se creara un archivo.")
			r := mkfileCmd.Bool("r", false, "Si existe, se pueden crear directorios padres.")
			size := mkfileCmd.Int("size", 0, "Tamaño del archivo a crear")
			cont := mkfileCmd.String("cont", "", "Ruta en la PC real donde se tomara un archivo.")

			mkfileCmd.Parse(args)

			if *path=="" {
				fmt.Println("Error: El parametro path debe ser obligatorio para mkfile.")
				
			}

			commands.ExecuteMkfile(*path, *r, *size, *cont)

		case "rep":
			repCmd := flag.NewFlagSet("rep", flag.ContinueOnError)
			name := repCmd.String("name", "", "Nombre del reporte a generar.")
			path := repCmd.String("path", "", "Ruta donde se creara el reporte.")
			id := repCmd.String("id", "", "Indica el ID de la particion.")
			path_file_ls := repCmd.String("path_file_ls", "", "Funciona con file y ls.")

			repCmd.Parse(args)

			if *name=="" || *path=="" || *id =="" {
				fmt.Println("Error: Los parametros name, path, id deben ser obligatorios para rep.")
			}

			if (*name=="ls" && *path_file_ls=="") || (*name=="file" && *path_file_ls=="") {
				fmt.Println("Error: El parametro path_file_ls es obligatorio cuando se utiliza file o ls")
			}

			commands.ExecuteRep(*name, *path, *id, *path_file_ls)


		default:
			fmt.Printf("Comando '%s' no reconocido.\n", command)
		}
	w.Close()

	// Lee la salida capturada
	var buf strings.Builder
	io.Copy(&buf, r)

	// Restaura stdout
	os.Stdout = oldStdout

	return buf.String()
}
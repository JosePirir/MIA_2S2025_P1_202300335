package main

// Importaciones necesarias para el funcionamiento del programa
import (
	"bufio"           // Paquete para leer la entrada de usuario línea por línea de manera eficiente
	"flag"            // Paquete para manejar flags/parámetros de línea de comandos
	"fmt"             // Paquete para formatear e imprimir texto en consola
	"proyecto1/commands" // Importa nuestro paquete personalizado que contiene los comandos disponibles
	"os"              // Paquete para interactuar con el sistema operativo (stdin, stderr, etc.)
	"strings"         // Paquete para manipular y modificar cadenas de texto
)

func main() {
	// bufio.Scanner lee línea por línea
	scanner := bufio.NewScanner(os.Stdin)

	// Bucle infinito para mantener el programa en ejecución hasta que el usuario decida salir
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break // Si no puede leer (EOF o error), sale del bucle
		}
		// Obtiene el texto de la línea leída, sin el carácter de nueva línea
		line := scanner.Text()
		// Verifica si el usuario quiere salir del programa
		// strings.ToLower() convierte a minúsculas para hacer la comparación insensible a mayúsculas
		if strings.ToLower(line) == "exit" {
			fmt.Println("Saliendo...")
			break // Termina el bucle y por tanto el programa
		}
		// strings.Fields() es más robusto que strings.Split() porque maneja múltiples espacios
		parts := strings.Fields(line)
		// Si el usuario solo presionó Enter (línea vacía), no hay nada que procesar
		if len(parts) == 0 {
			continue // Vuelve al inicio del bucle para mostrar el prompt nuevamente
		}

		// La primera palabra siempre es el comando a ejecutar
		command := strings.ToLower(parts[0]) // Convertir a minúsculas para comparación
		// El resto de las palabras son los argumentos y flags del comando
		args := parts[1:] // Slice que incluye desde el índice 1 hasta el final
		// Normalizar cada argumento a minúsculas
		for i, arg := range args {
			// Si es un flag tipo --X=valor
			if strings.HasPrefix(arg, "-") {
				parts := strings.SplitN(arg, "=", 2)
				parts[0] = strings.ToLower(parts[0]) // nombre del flag a minúsculas
				if len(parts) == 2 {
					args[i] = parts[0] + "=" + parts[1] // conserva el valor tal cual
				} else {
					args[i] = parts[0]
				}
			} else {
				// Si no es flag (ej: un path con mayúsculas), no lo tocamos
				args[i] = arg
			}
		}

		// Switch para determinar qué comando ejecutar
		switch command {
		case "mkdisk":
			// Crear un nuevo conjunto de flags específico para el comando mkdisk
			// flag.ExitOnError hace que el programa termine si hay un error en los flags
			mkdiskCmd := flag.NewFlagSet("mkdisk", flag.ExitOnError)

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
				continue // Vuelve al inicio del bucle sin ejecutar el comando
			}

			// El parámetro size debe ser positivo y mayor que cero
			if *size <= 0 {
				fmt.Println("Error: el parámetro -size es obligatorio y debe ser positivo.")
				continue // Vuelve al inicio del bucle sin ejecutar el comando
			}

			// Si todas las validaciones pasan, ejecuta el comando mkdisk
			// Pasa los valores desreferenciados (con *) a la función
			commands.ExecuteMkdisk(*size, *unit, *fit, *path)

			case "rmdisk":
			rmdiskCmd := flag.NewFlagSet("rmdisk", flag.ExitOnError)
			path := rmdiskCmd.String("path", "", "Ruta del disco a eliminar.")
			rmdiskCmd.Parse(args)

			if *path == "" {
				fmt.Println("Error: el parámetro -path es obligatorio para rmdisk.")
				continue
			}
			commands.ExecuteRmdisk(*path)

		default:
			// Si el comando no es reconocido, mostrar mensaje de error
			fmt.Printf("Comando '%s' no reconocido.\n", command)
		}
		
	}

	

	// Manejo de errores del scanner
	// scanner.Err() retorna cualquier error que haya ocurrido durante la lectura
	if err := scanner.Err(); err != nil {
		// fmt.Fprintln() imprime en stderr (salida de error estándar) en lugar de stdout
		fmt.Fprintln(os.Stderr, "Error leyendo la entrada:", err)
	}
}
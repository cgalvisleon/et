package logs

import (
	"fmt"
	"io"
	"os"
	"os/signal"
)

func StdOut() error {
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	// Guarda la salida original para restaurarla luego
	originalStdout := os.Stdout

	// Redirige stdout a la parte de escritura de la tubería
	os.Stdout = w

	// Usa un gorro para leer de la tubería en paralelo
	// var wg sync.WaitGroup
	// wg.Add(1)
	go func() {
		// defer wg.Done()
		// Lee desde la parte de lectura de la tubería
		var buf [1024]byte
		n, err := r.Read(buf[:])
		if err != nil && err != io.EOF {
			fmt.Println("Error reading from pipe:", err)
			return
		}
		// Imprime lo que se ha capturado
		fmt.Println("Captured stdout:", string(buf[:n]))
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// Imprime algo al stdout (que ahora va a la tubería)
	fmt.Println("Hello, world!")

	// Cierra el canal de escritura para terminar la lectura
	w.Close()

	// Espera a que el gorro termine
	// wg.Wait()

	// Restaura la salida estándar original
	os.Stdout = originalStdout

	// Ahora puedes imprimir al stdout nuevamente
	fmt.Println("This is back to the original stdout.")

	return nil
}

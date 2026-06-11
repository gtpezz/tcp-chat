package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer conn.Close()

	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			fmt.Println("\nСоединение с сервером потеряно. Нажмите Enter для выхода")
			time.Sleep(5 * time.Second)
			os.Exit(1)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()

		_, err := conn.Write([]byte(text + "\n"))
		if err != nil {
			fmt.Println("Connection lost")
			os.Exit(1)
		}

	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}

}

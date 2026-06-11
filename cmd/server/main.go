package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tcp-chat/internal"
	"time"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	server := internal.NewServer()
	go server.Run()

	fmt.Println("Server start on " + listener.Addr().String())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Сервер закрывается ...")

		server.Broadcast <- "Система: сервер закроется через 10 секунд"

		time.Sleep(10 * time.Second)

		listener.Close()

		server.Mtx.Lock()
		for _, client := range server.Clients {
			client.Conn.Close()
		}
		server.Mtx.Unlock()
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		client := &internal.Client{
			Name: "",
			Conn: conn,
			Send: make(chan string, 256),
		}

		server.Register <- client

		go handleClient(conn, client, server)
	}
}

func handleClient(
	conn net.Conn,
	client *internal.Client,
	server *internal.Server,
) {
	defer func() {
		server.Unregister <- conn
		conn.Close()
	}()

	client.Send <- "Добро пожаловать в чат"
	client.Send <- "Введите ваше имя"

	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	scanner := bufio.NewScanner(conn)

	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				client.Send <- fmt.Sprintf("Ошибка: %v", err)
			} else {
				client.Send <- "Время на ввод имени истекло"
			}
			return
		}

		name := strings.TrimSpace(scanner.Text())

		if !isValidName(name) {
			client.Send <- "Некорректное имя. Используйте 3-20 символов (буквы, цифры, _). Попробуйте снова:"
			continue
		}

		if server.IsNameTaken(name) {
			client.Send <- "Система: Имя уже занято, выберите другое. Попробуйте снова:"
			continue
		}

		client.Name = name
		break
	}

	conn.SetReadDeadline(time.Time{})

	welcomeMsg := fmt.Sprintf("[%s] Система: Добро пожаловать, %s!",
		time.Now().Format("15:04:05"), client.Name)

	client.Send <- welcomeMsg

	clientsList := server.GetClientList()
	if clientsList != "" {
		client.Send <- fmt.Sprintf("[%s] Система: Пользователи онлайн: %s",
			time.Now().Format("15:04:05"), clientsList)
	}

	server.Broadcast <- fmt.Sprintf("[%s] Система: %s присоединился к чату",
		time.Now().Format("15:04:05"), client.Name)

	for scanner.Scan() {
		text := scanner.Text()

		if text == "/exit" {
			break
		}

		if strings.HasPrefix(text, "/") {
			handleCommand(text, client, server)
		} else {
			server.Broadcast <- fmt.Sprintf("[%s] %s: %s",
				time.Now().Format("15:04:05"), client.Name, text)
		}
	}

	if client.Name != "" {
		server.Broadcast <- fmt.Sprintf("[%s] Система: %s покинул чат",
			time.Now().Format("15:04:05"), client.Name)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
}

func handleCommand(text string, client *internal.Client, server *internal.Server) {
	switch {
	case text == "/users":
		clientsList := server.GetClientList()
		if clientsList == "" {
			client.Send <- "Других пользователей нет"
		} else {
			client.Send <- fmt.Sprintf("Пользователи онлайн: %s", clientsList)
		}

	case strings.HasPrefix(text, "/nick "):
		newName := strings.TrimSpace(strings.TrimPrefix(text, "/nick "))
		if !isValidName(newName) {
			client.Send <- "Некорректное имя. Используйте 3-20 символов (буквы, цифры, _)"
		} else if server.IsNameTaken(newName) {
			client.Send <- "Имя уже занято"
		} else {
			oldName := client.Name
			client.Name = newName
			server.Broadcast <- fmt.Sprintf("[%s] Система: %s сменил имя на %s",
				time.Now().Format("15:04:05"), oldName, newName)
		}

	case strings.HasPrefix(text, "/msg "):
		parts := strings.SplitN(text, " ", 3)
		if len(parts) < 3 {
			client.Send <- "Использование: /msg <пользователь> <сообщение>"
		} else {
			targetName := parts[1]
			message := parts[2]
			server.SendPrivateMessage(client.Name, targetName, message)
		}
	case text == "/help":
		help := "Доступные команды:\n" +
			"/nick <имя> - сменить никнейм\n" +
			"/users - показать список пользователей\n" +
			"/msg <пользователь> <сообщение> - отправить личное сообщение\n" +
			"/help - показать эту справку\n" +
			"/exit - выйти из чата"
		client.Send <- help

	default:
		client.Send <- "Неизвестная команда. Введите /help для списка команд"
	}
}

func isValidName(name string) bool {
	if len(name) < 3 || len(name) > 20 {
		return false
	}

	lowerName := strings.ToLower(name)
	if lowerName == "система" || lowerName == "system" || lowerName == "admin" {
		return false
	}

	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	return true
}

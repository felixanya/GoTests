package gameserver

import (
	"log"
	"net"
)

type Server struct {
	listener       *net.TCPListener
	listenerExitCh chan bool
	loopExitCh     chan bool
	gameRooms      map[uint32]*GameRoom
	removeRoomCh   chan *GameRoom
	makeClientCh   chan *net.TCPConn
}

// Создание нового сервера
func NewServer() *Server {
	server := Server{
		listener:       nil,
		listenerExitCh: make(chan bool),
		loopExitCh:     make(chan bool),
		gameRooms:      make(map[uint32]*GameRoom),
		removeRoomCh:   make(chan *GameRoom),
		makeClientCh:   make(chan *net.TCPConn),
	}
	return &server
}

func (server *Server) ExitServer() {
	server.exitAsyncSocketListener()
	server.exitMainLoop()
}

func (server *Server) StartListen() {
	server.asyncSocketAcceptListener()
	server.mainLoop()
}

func (server *Server) DeleteRoom(room *GameRoom) {
	server.removeRoomCh <- room
}

// Обработка входящих подключений
func (server *Server) asyncSocketAcceptListener() {
	address, err := net.ResolveTCPAddr("tcp", ":9999")
	if err != nil {
		log.Println("Server address resolve error")
		server.ExitServer()
		return
	}

	// Создание листенера
	createdListener, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Printf("Server listener start error: %s\n", err)
		server.ExitServer()
		return
	}

	// Сохраняем листенер для возможности закрытия
	server.listener = createdListener

	// Функция-цикл обработки входящих подключений
	loopFunction := func() {
		for {
			select {
			case <-server.listenerExitCh:
				server.listener.Close()
				log.Print("Socket listener exit") // Наш лиснер закрылся и надо будет выйти из цикла
				return

			default:
				// Ожидаем новое подключение
				c, err := server.listener.AcceptTCP()
				if err != nil {
					log.Printf("Accept error: %s\n", err) // Наш лиснер закрылся и надо будет выйти из цикла
					continue
				}

				log.Printf("Connection accepted\n")
				c.SetKeepAlive(true)
				c.SetNoDelay(true)

				// Раз появилось новое соединение - запускаем его в работу с отдельной горутине
				server.makeClientCh <- c
			}
		}
	}

	go loopFunction()
}

// Выход из листенера
func (server *Server) exitAsyncSocketListener() {
	if server.listener != nil {
		server.listener.Close()
	}
	server.listenerExitCh <- true
}

// Основная функция прослушивания
func (server *Server) mainLoop() {
	loopFunction := func() {
		for {
			select {
			// Обрабатываем новое подключение
			case connection := <-server.makeClientCh:
				log.Printf("Make client call\n")

				roomFound := false
				for _, gameRoom := range server.gameRooms {
					if gameRoom.GetIsFull() == false {
						gameRoom.AddClientForConnection(connection)
						roomFound = true
						break
					}
				}
				// Не нашли подходящей свободной комнаты
				if roomFound == false {
					newGameRoom := NewGameRoom(server)
					server.gameRooms[newGameRoom.roomId] = newGameRoom

					newGameRoom.StartLoop()

					newGameRoom.AddClientForConnection(connection)
				}

			// Обработка удаления комнаты
			case room := <-server.removeRoomCh:
				delete(server.gameRooms, room.roomId)

			// Завершение работы
			case <-server.loopExitCh:
				log.Print("Main loop exit") // Наш лиснер закрылся и надо будет выйти из цикла
				return
			}
		}
	}
	go loopFunction()
}

func (server *Server) exitMainLoop() {
	server.loopExitCh <- true
}

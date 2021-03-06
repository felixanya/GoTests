package gameserver

import (
	"container/list"
	"encoding/binary"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const UPDATE_QUEUE_SIZE = 100

// Variables
var MAX_ID uint32 = 0

type ClientShootUpdateResult struct {
	clientID uint32
	bullet   *Bullet
	client   *Client
}

type ClientPositionInfo struct {
	clientID uint32
	x        int16
	y        int16
	size     uint8
	client   *Client
}

// Структура клиента
type Client struct {
	server       *Server
	connection   *net.TCPConn
	id           uint32
	state        *ClientState
	mutex        sync.RWMutex
	uploadDataCh chan []byte
	exitReadCh   chan bool
	exitWriteCh  chan bool
}

// NewClient ... Конструктор
func NewClient(connection *net.TCPConn, server *Server) *Client {
	if connection == nil {
		panic("No connection")
	}
	if server == nil {
		panic("No game server")
	}

	// Увеличиваем id
	curId := atomic.AddUint32(&MAX_ID, 1)

	// Состояние для отгрузки клиенту
	clientState := NewState(curId, int16(rand.Int()%200+100), int16(rand.Int()%200+100))

	// Конструируем клиента и его каналы
	uploadDataCh := make(chan []byte, UPDATE_QUEUE_SIZE) // В канале апдейтов может накапливаться максимум 1000 апдейтов
	exitReadCh := make(chan bool, 1)
	exitWriteCh := make(chan bool, 1)

	return &Client{
		server:       server,
		connection:   connection,
		id:           curId,
		state:        clientState,
		mutex:        sync.RWMutex{},
		uploadDataCh: uploadDataCh,
		exitReadCh:   exitReadCh,
		exitWriteCh:  exitWriteCh,
	}
}

func (client *Client) GetCurrentStateData() ([]byte, error) {
	client.mutex.RLock()
	stateData, err := client.state.ConvertToBytes()
	client.mutex.RUnlock()
	return stateData, err
}

func (client *Client) UpdateCurrentState(delta float64, worldSizeX, worldSizeY uint16) (bool, []ClientShootUpdateResult, ClientPositionInfo) {
	maxX := float64(worldSizeX)
	maxY := float64(worldSizeY)

    hasNews := false
	bullets := []ClientShootUpdateResult{}
	deleteBullets := []*list.Element{}

	client.mutex.Lock()

	// Position info
	positionInfo := ClientPositionInfo{
        clientID: client.id,
		x:      client.state.X,
		y:      client.state.Y,
		size:   client.state.Size,
		client: client,
	}

	// Bullets
	if client.state.Status != CLIENT_STATUS_FAIL {
		// обновление позиций пуль с удалением старых
		it := client.state.Bullets.Front()
		for i := 0; i < client.state.Bullets.Len(); i++ {

			bul := it.Value.(*Bullet)
			bul.WorldTick(delta)

			// Проверяем пулю на выход из карты
			if (bul.X > 0) && (bul.X < maxX) && (bul.Y > 0) && (bul.Y < maxY) {
				clientBulletPair := ClientShootUpdateResult{
                    clientID: client.id,
					client: client,
					bullet: bul,
				}
				bullets = append(bullets, clientBulletPair)
                hasNews = true
			} else {
				deleteBullets = append(deleteBullets, it)
                hasNews = true
			}

			it = it.Next()
		}
		// Удаление старых
		for _, it := range deleteBullets {
			client.state.Bullets.Remove(it)
		}
	}
	client.mutex.Unlock()
	return hasNews, bullets, positionInfo
}

func (client *Client) IncreaseFrag(bullet *Bullet) {
	client.mutex.Lock()
    {
        // Frag increase
        client.state.Frags++
        // Delete bullet
        it := client.state.Bullets.Front()
        for i := 0; i < client.state.Bullets.Len(); i++ {
            bul := it.Value.(*Bullet)
            if bul.ID == bullet.ID {
                client.state.Bullets.Remove(it)
                break
            }
            it = it.Next()
        }
    }
	client.mutex.Unlock()
}

func (client *Client) SetFailStatus() {
	client.mutex.Lock()
	client.state.Status = CLIENT_STATUS_FAIL
	client.mutex.Unlock()
}

// Пишем сообщение клиенту
func (client *Client) QueueSendGameState(gameState []byte) {
	// Если очередь превышена - считаем, что юзер отвалился
	if len(client.uploadDataCh)+1 > UPDATE_QUEUE_SIZE {
		log.Printf("Queue full for state %d", client.id)
		return
	} else {
		client.uploadDataCh <- gameState
	}
}

// Пишем сообщение клиенту только с его состоянием
func (client *Client) QueueSendCurrentClientState() {
	// Если очередь превышена - считаем, что юзер отвалился
	if len(client.uploadDataCh)+1 > UPDATE_QUEUE_SIZE {
		log.Printf("Queue full for state %d", client.id)
		return
	} else {
		client.mutex.RLock()
		data, err := client.state.ConvertToBytes()
		client.mutex.RUnlock()

		if err != nil {
			log.Printf("State upload error for state %d: %s\n", client.id, err)
		}

		client.uploadDataCh <- data
	}
}

// Запускаем ожидания записи и чтения (блокирующая функция)
func (client *Client) StartLoop() {
	go client.loopWrite() // в отдельной горутине
	go client.loopRead()
}

func (client *Client) StopLoop() {
	client.exitWriteCh <- true
	client.exitReadCh <- true
    client.connection.Close()
}

// Ожидание записи
func (client *Client) loopWrite() {
	//log.Println("StartSyncListenLoop write to state:", state.id)
	for {
		select {
		// Отправка записи клиенту
		case payloadData := <-client.uploadDataCh:
			// Размер данных
			dataBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(dataBytes, uint32(len(payloadData)))

			// Данные для отправки
			sendData := append(dataBytes, payloadData...)

			// Таймаут
			timeout := time.Now().Add(30 * time.Second)
			client.connection.SetWriteDeadline(timeout)

			// Отсылаем
			writenCount, err := client.connection.Write(sendData)
			if (err != nil) || (writenCount < len(sendData)) {
				client.server.DeleteClient(client)
                client.connection.Close()
				client.exitReadCh <- true // Выход из loopRead

				if err != nil {
					log.Printf("LoopWrite exit by ERROR (%s), clientId = %d\n", err, client.id)
				} else if writenCount < len(sendData) {
					log.Printf("LoopWrite exit by less bytes - %d from %d, clientId = %d\n", writenCount, len(sendData), client.id)
				}
				return
			}

		// Получение флага выхода из функции
		case <-client.exitWriteCh:
			log.Println("LoopWrite exit, clientId =", client.id)
			return
		}
	}
}

// Ожидание чтения
func (client *Client) loopRead() {
	//log.Println("Listening read from state")
	for {
		select {
		// Получение флага выхода
		case <-client.exitReadCh:
			log.Println("LoopRead exit, clientId =", client.id)
			return

		// Чтение данных из сокета
		default:
			// Ожидается, что за 5 минут что-то придет, иначе - это отвал
			timeout := time.Now().Add(5 * time.Minute)
			client.connection.SetReadDeadline(timeout)

			// Размер данных
			dataSizeBytes := make([]byte, 4)
			readCount, err := client.connection.Read(dataSizeBytes)

			// Ошибка чтения данных
			if (err != nil) || (readCount < 4) {
				client.server.DeleteClient(client)
                client.connection.Close()
				client.exitWriteCh <- true // для метода loopWrite, чтобы выйти из него

				if err == io.EOF {
					log.Printf("LoopRead exit by disconnect, clientId = %d\n", client.id)
				} else if err != nil {
					log.Printf("LoopRead exit by ERROR (%s), clientId = %d\n", err, client.id)
				} else if readCount < 4 {
					log.Printf("LoopRead exit - read less 8 bytes (%d bytes), clientId = %d\n", readCount, client.id)
				}
				return
			}
			dataSize := int(binary.BigEndian.Uint32(dataSizeBytes))

			// Ожидается, что будут данные в течении 30 секунд - иначе отвал
			timeout = time.Now().Add(30 * time.Second)
			client.connection.SetReadDeadline(timeout)

			// Данные
			data := make([]byte, dataSize)
			dataOffset := 0
			readingSuccess := false
			for {
				readCount, err = client.connection.Read(data[dataOffset:])
				dataOffset += readCount

				// Ошибка чтения данных
				if err != nil {
					client.server.DeleteClient(client)
                    client.connection.Close()
					client.exitWriteCh <- true // для метода loopWrite, чтобы выйти из него

					if err == io.EOF {
						log.Printf("LoopRead exit by disconnect, clientId = %d\n", client.id)
					} else if err != nil {
						log.Printf("LoopRead exit by ERROR (%s), clientId = %d\n", err, client.id)
					}
					return
				} else if readCount == 0 {
					client.server.DeleteClient(client)
                    client.connection.Close()
					client.exitWriteCh <- true // для метода loopWrite, чтобы выйти из него

					log.Printf("LoopRead exit by disconnect (read 0 bytes), clientId = %d\n", client.id)

				} else if dataOffset == dataSize {
					readingSuccess = true
					break
				}
			}

			if readingSuccess {
				if IsClientCommandData(data) {
					// Декодирование
					command, err := NewClientCommand(data)

					if (err == nil) && (command.ID == client.id) {
						// Обновление состояния
						client.mutex.Lock()
						{
							client.state.X = command.X
							client.state.Y = command.Y
							client.state.Angle = command.Angle
							// Дополнительные действия
							switch command.Type {
							case CLIENT_COMMAND_TYPE_MOVE:
								break
							case CLIENT_COMMAND_TYPE_SHOOT:
								bullet := NewBullet(client.state.X, client.state.Y, int16(client.state.Size)/2, client.state.Angle)
								client.state.Bullets.PushBack(bullet)
								break
							}
						}
						client.mutex.Unlock()

						// Запрашиваем отправку обновления состояния всем
						client.server.QueueSendAllNewState()
					} else {
						client.server.DeleteClient(client)
                        client.connection.Close()
						client.exitWriteCh <- true // для метода loopWrite, чтобы выйти из него

						log.Printf("Wrong command data, clientId = %d, %v\n", client.id, command)
					}
				} else {
					client.server.DeleteClient(client)
                    client.connection.Close()
					client.exitWriteCh <- true // для метода loopWrite, чтобы выйти из него

					log.Printf("Is not client command data, clientId = %d\n", client.id)
				}
			}
		}
	}
}

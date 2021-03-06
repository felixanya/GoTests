package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

func rawClient() {
	// Определяем адрес
	address, err := net.ResolveUDPAddr("udp", "devnulpavel.ddns.net:9999") // devnulpavel.ddns.net 	192.168.1.3
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Resolved ip address: %s\n", address)

	// Подключение к серверу
	c, err := net.DialUDP("udp", nil, address)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()

	readErrorCounter := 0

	const dataSize = 1024
    const timeBegin = 0
    const counterBegin = 200
    data := make([]byte, dataSize)
	var counter uint64 = 0

	const requestsCount = 100
	startTime := time.Now()
	for i := 0; i < requestsCount; i++ {
		sendTime := uint64(time.Now().UnixNano())
		binary.BigEndian.PutUint64(data[timeBegin:timeBegin+8], sendTime)
		binary.BigEndian.PutUint64(data[counterBegin:counterBegin+8], counter)

		// Пытаемся записать данные
		c.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
		currentWritten, err := c.Write(data)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				fmt.Printf("Is WRITE timeout error: %s\n", err)
				continue
			} else {
				fmt.Println(err)
				return
			}
			return
		} else if currentWritten < dataSize {
			fmt.Printf("Written less bytes - %d\n", currentWritten)
			return
		}

		// теперь читаем
		c.SetReadDeadline(time.Now().Add(5000 * time.Millisecond))
		receivedCount, senderAddress, err := c.ReadFromUDP(data)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				fmt.Printf("Is READ timeout error: %s\n", err)
				readErrorCounter++
				if readErrorCounter > 5 {
					fmt.Println("Disconnected by many read timeouts")
					return
				} else {
					continue
				}
			} else {
				fmt.Printf("Read error: %s\n", err)
				return
			}
		} else if receivedCount == 0 {
			fmt.Println("Disconnected")
			return
		} else if receivedCount < dataSize {
			fmt.Printf("Received less data size - %d\n", receivedCount)
		}
		// Reset read counter error
		readErrorCounter = 0

		receivedSendTimeUint64 := binary.BigEndian.Uint64(data[timeBegin:timeBegin+8])
		receivedCounterUint64 := binary.BigEndian.Uint64(data[counterBegin:counterBegin+8])

		if receivedCounterUint64 != counter {
			fmt.Println("Receive counter error")
			continue
		}

		counter++

		// Ping
		receivedSendTime := time.Unix(0, int64(receivedSendTimeUint64))
		ping := float64(time.Now().Sub(receivedSendTime).Nanoseconds()) / 1000.0 / 1000.0
		fmt.Printf("Ping = %fms, from adress: %s\n", ping, senderAddress)

		//time.Sleep(1000 * time.Millisecond)
	}

    endTime := time.Now()
    duration := endTime.Sub(startTime).Seconds()

    requestsPerSec := requestsCount/duration
    fmt.Printf("Requests per sec value: %f", requestsPerSec)
}

func main() {
	rawClient()

	var input string
	fmt.Scanln(&input)
}

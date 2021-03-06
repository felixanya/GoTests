package main

import "fmt"
import "net"
import "time"
import "encoding/gob"
import "encoding/json"
import (
    "./request"
    "log"
)

func client() {
    // Подключение к серверу
    c, err := net.Dial("tcp", "127.0.0.1:9999")
    if err != nil {
        fmt.Println(err)
        return
    }
    defer c.Close()

    encoder := gob.NewEncoder(c)
    decoder := gob.NewDecoder(c) 

    for {
        // Отправка сообщения серверу
        msg := "Hello World"
        fmt.Println("Sending:", msg)

        err = encoder.Encode(msg)
        if err != nil {
            fmt.Println(err)
            return
        }

        time.Sleep(time.Millisecond * 200)

        // Получаем сообщение
        err := decoder.Decode(&msg)
        if err != nil {
            fmt.Println(err)
            return
        } else if msg == "ok" {
            fmt.Println("Sent")
        } else{
            fmt.Println("Sending error")
            return
        }
    }
}

func jsonClient() {
    // Подключение к серверу
    c, err := net.Dial("tcp", "127.0.0.1:9999")
    if err != nil {
        fmt.Println(err)
        return
    }
    defer c.Close()

    encoder := json.NewEncoder(c)
    decoder := json.NewDecoder(c)

    for {
        // Отправка сообщения серверу
        sendMsg := request.NewRequest(request.REQUEST_TYPE_GET_SERVER_STATE)
        sendMsg.Params["param1"] = "test1"
        sendMsg.Params["param2"] = float64(20)
        sendMsg.Params["param3"] = float64(1.56)
        sendMsg.Params["param4"] = []float64{100, 200, 300}
        sendMsg.SubData = request.RequstSubData{"test1", "test2"}

        {
            // Отладочный вывод
            jsonBytes, err := json.Marshal(sendMsg)
            if err == nil {
                fmt.Println("Send json:", string(jsonBytes))
            }
        }

        err = encoder.Encode(sendMsg)
        if err != nil {
            fmt.Println(err)
            return
        }

        time.Sleep(time.Millisecond * 500)

        // Получаем сообщение
        receivedMsg := request.Request{}
        err := decoder.Decode(&receivedMsg)
        if err != nil {
            fmt.Println(err)
            return
        } else{
            jsonBytes, err := json.Marshal(receivedMsg)
            if err == nil {
                fmt.Println("Rceived json:", string(jsonBytes), "\n")
            }
            //fmt.Println("Response:", receivedMsg)
        }
    }
}

func rawClient()  {
    // Подключение к серверу
    c, err := net.Dial("tcp", "127.0.0.1:9999")
    if err != nil {
        fmt.Println(err)
        return
    }
    defer c.Close() // Отложеннное закрытие при выходе

    // TODO: в одном блоке данных могут быть получены сразу 2 сообщения

    // Бесконечный цикл записи
    for {
        timeVal := time.Now().Add(5 * time.Minute)
        c.SetDeadline(timeVal)
        c.SetWriteDeadline(timeVal)
        c.SetReadDeadline(timeVal)

        testData := []byte("Test message");
        testData = append(testData, 0)
        testDataSize := len(testData)

        writeSuccess := false
        totalWrittenBytes := 0
        for totalWrittenBytes < testDataSize {
            currentWritten, err := c.Write(testData[writtenBytes:])
            if err != nil {
                log.Println(err)
                break;
            }

            totalWrittenBytes += currentWritten
            if totalWrittenBytes == testDataSize {
                writeSuccess = true
                break
            }
        }

        if writeSuccess {
            fmt.Println("Write success")

            // Теперь очередь чтения??
            getData := make([]byte, 0)
            _, err = c.Read(getData)
            if err != nil {
                return
            }
        }else{
            fmt.Println("Write failed")
            break
        }
    }
}

func main() {
    rawClient()

    var input string
    fmt.Scanln(&input)
}
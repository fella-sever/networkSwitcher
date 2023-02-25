package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"networkSwitcher/domain"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

func main() {
	//networkSwitchModeChan := make(chan string, 2)
	var set domain.MetricsCount
	// задаем дефолтное значение параметров запуска утилиты
	set.RttSettings = 80               // предел задержки по сети
	set.PacketLossSettings = 60        // предел потери пакетов
	set.PingerCount = 10               // сколько пакетов надо плюнуть
	set.PingerInterval = 20            // за какое время это пакеты на выплюнуть
	set.NetworkSwitchMode = "auto"     // автоматический режим переключения сети по умолчанию
	err := domain.IpTablesSwitchMain() // стартовое переключение на основной канал связи
	if err != nil {
		fmt.Println("main: iptablesSwitchMain: main network is unreachable")
	}
	wg := sync.WaitGroup{}
	wg.Add(3)
	r := gin.Default()

	// запуск сервера
	go func() {
		r.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, set)
		})
		r.POST("/set_threshold", func(c *gin.Context) {
			var newSettings domain.MetricsSetDto
			if err := c.BindJSON(&newSettings); err != nil {
				return
			}

			set.PacketLossSettings = newSettings.PacketLoss
			set.RttSettings = newSettings.RttSettings
			set.PingerCount = newSettings.PingerCount
			set.PingerInterval = newSettings.PingerInterval
			c.IndentedJSON(http.StatusCreated, set)

		})
		r.POST("/set_network_mode", func(c *gin.Context) {
			var networkSwitchMode domain.NetworkSwitchSettingsDTO
			if err := c.BindJSON(&networkSwitchMode); err != nil {
				return
			}
			set.NetworkSwitchMode = networkSwitchMode.NetworkSwitchMode
			c.IndentedJSON(http.StatusAccepted,
				"network mode: "+set.NetworkSwitchMode)
		})
		r.Run()
		wg.Done()
	}()
	//запуск сканирования состояния сети
	go func() {
		for {
			ping, err := exec.Command("ping", "-i 0.2", "-c 10", "8.8.8.8").Output()
			if err != nil {
				fmt.Println(err)
			}
			stringPing := string(ping)
			packetLoss := strings.Split(stringPing, "\n")
			rttRow := packetLoss[len(packetLoss)-2]
			packetLossRow := packetLoss[len(packetLoss)-3]
			splittedRttRow := strings.Split(rttRow, "/")
			splittedPacketLossRow := strings.Split(packetLossRow, ",")
			finalRtt, err := strconv.ParseFloat(splittedRttRow[4], 64)
			if err != nil {
				fmt.Println(err)
			}
			//fmt.Println(string(splittedPacketLossRow[2]))
			finalPackeLoss, lossErr := strconv.Atoi(string(
				splittedPacketLossRow[2][1]))
			if err != nil {
				fmt.Println(lossErr)
			}
			set.Rtt = finalRtt
			set.PacketLoss = finalPackeLoss * 10

		}
		wg.Done()
	}()

	wg.Wait()

}

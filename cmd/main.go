package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"log"
	"net/http"
	"networkSwitcher/domain"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

func main() {
	networkModeChan := make(chan string, 2)
	rttCurrent := make(chan float64)
	packetLossCurrent := make(chan float64)
	// начальная проверка интерфейсов
	var set domain.MetricsCount
	interfaceErr := set.CheckInterfaceIsAlive("main")
	if interfaceErr != nil {
		log.Println("check interfaces: ", interfaceErr)
	}
	interfaceErr = set.CheckInterfaceIsAlive("reserve")
	if interfaceErr != nil {
		log.Println("check interfacres: ", interfaceErr)
	}
	// валидатор для роутов
	validate := validator.New()
	// дефолтное значение параметров запуска утилиты
	set.RttSettings = 100          // предел задержки по сети
	set.PacketLossSettings = 60    // предел потери пакетов
	set.PingerCount = 10           // сколько пакетов надо плюнуть
	set.PingerInterval = 20        // за какое время это пакеты на выплюнуть
	set.NetworkSwitchMode = "auto" // автоматический режим переключения сети по умолчанию
	networkModeChan <- "auto"
	wg := sync.WaitGroup{}
	wg.Add(4)
	r := gin.Default()
	// запск сервера
	go func() {
		r.GET("/get_info", func(c *gin.Context) {
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
			errs := validate.Struct(&networkSwitchMode)
			if errs != nil {
				c.IndentedJSON(http.StatusBadRequest, "bad validation:")
			} else {
				set.NetworkSwitchMode = networkSwitchMode.NetworkSwitchMode
				networkModeChan <- networkSwitchMode.NetworkSwitchMode
				c.IndentedJSON(http.StatusAccepted,
					"network mode: "+set.NetworkSwitchMode)
			}

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
			finalPacketLoss, lossErr := strconv.ParseFloat(string(
				splittedPacketLossRow[2][1]), 64)
			if err != nil {
				fmt.Println(lossErr)
			}

			set.Rtt = finalRtt
			set.PacketLoss = finalPacketLoss * 10
			rttCurrent <- finalRtt
			packetLossCurrent <- finalPacketLoss * 10

		}
		wg.Done()
	}()
	// переключатель режима сети
	go func() {
		for {
			switch <-networkModeChan {
			case "auto":
				err := set.NetworkAutoSwitch(rttCurrent, packetLossCurrent)
				if err != nil {
					fmt.Println("autoSwitch err: ", err)
				}
			case "reserve":
				err := domain.IpTablesSwitchReseve()
				if err != nil {
					fmt.Println("switch to reserve iptables mode err: ", err)
				}
			case "main":
				err := domain.IpTablesSwitchMain()
				if err != nil {
					fmt.Println("switch to reserve iptables mode err: ", err)
				}
			}
		}
	}()

	wg.Wait()

}

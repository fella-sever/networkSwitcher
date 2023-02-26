package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"net/http"
	"networkSwitcher/domain"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

func main() {
	var log = logrus.New()
	logFile, createLogerr := os.OpenFile("networkSwitcherLog.txt",
		os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
	if createLogerr != nil {
		fmt.Println(createLogerr)
	}
	log.Out = logFile
	// переключалка сети - устанавливает режим работы свитчера в зависимости от того
	// что выбрал пользователь в эндпоинте. Ниже по дефолту в канал пишется "auto"
	// для работы свитчера в режиме авто по умолчанию
	networkModeChan := make(chan string, 2)
	// синхронизаторы служат для торможения и синхронизации суперцикла в рутине
	// переключения сети в автоматическом режиме. В случае неиспользования суперцикл
	// в рутине гонит со страшной скоростью, забирая на себя все вычсилительные
	// ресурсы ядра
	rttCurrent := make(chan float64)        // синхронизатор по ртт
	packetLossCurrent := make(chan float64) // синхронизатор по потере пакетов

	// начальная проверка интерфейсов на доступность их в системе. Если основной интер
	// фейл недоступен, то происходит свитч на резерв
	var set domain.MetricsCount
	interfaceErr := set.CheckInterfaceIsAlive("main")
	if interfaceErr != nil {
		log.Println("check interfaces: ", interfaceErr)
	}
	interfaceErr = set.CheckInterfaceIsAlive("reserve")
	if interfaceErr != nil {
		log.Println("check interfacres: ", interfaceErr)
	}
	// валидатор для роутов, конкретнее для валидации поля выбора режима работы сети
	validate := validator.New()
	// дефолтное значение параметров запуска утилиты
	set.RttSettings = 100          // предел задержки по сети
	set.PacketLossSettings = 60    // предел потери пакетов
	set.PingerCount = 10           // сколько пакетов надо плюнуть
	set.PingerInterval = 20        // за какое время это пакеты на выплюнуть
	set.NetworkSwitchMode = "auto" // автоматический режим переключения сети по умолчанию
	networkModeChan <- "auto"
	log.Infof("starting with default parameters")
	wg := sync.WaitGroup{}
	wg.Add(4)
	// запуск сервера
	r := gin.Default()
	go func() {
		// роут для получения пользователем информации о системе и насройках
		r.GET("/get_info", func(c *gin.Context) {
			c.JSON(http.StatusOK, set)
		})
		// роут для установки пороговых значений ртт и потери пакетов
		// для установки режимов пинга
		// TODO пока еще не прокинул настройки пинга в функцию самого пинга
		// надо доделать
		r.POST("/set_threshold", func(c *gin.Context) {
			var newSettings domain.MetricsUserSetDto
			if err := c.BindJSON(&newSettings); err != nil {
				return
			}

			set.PacketLossSettings = newSettings.PacketLoss
			set.RttSettings = newSettings.RttSettings
			set.PingerCount = newSettings.PingerCount
			set.PingerInterval = newSettings.PingerInterval
			c.IndentedJSON(http.StatusCreated, set)
			log.Infof("changed thresholds: packetLoss settings - %0.2f,"+
				"rtt settings - %0.2f, pinger count - %d, pinger interval - %d",
				newSettings.PacketLoss, newSettings.RttSettings,
				newSettings.PingerCount, newSettings.PingerInterval)

		})
		// выбор режима сети
		r.POST("/set_network_mode", func(c *gin.Context) {
			var networkSwitchMode domain.NetworkSwitchSettingsUserSetDTO
			if err := c.BindJSON(&networkSwitchMode); err != nil {
				return
			}
			errs := validate.Struct(&networkSwitchMode)
			if errs != nil {
				c.IndentedJSON(http.StatusBadRequest, "bad validation:")
				log.Infof("bad validation of network: %s",
					networkSwitchMode.NetworkSwitchMode)
			} else {
				set.NetworkSwitchMode = networkSwitchMode.NetworkSwitchMode
				networkModeChan <- networkSwitchMode.NetworkSwitchMode
				c.IndentedJSON(http.StatusAccepted,
					"network mode: "+set.NetworkSwitchMode)
				log.Infof("network mode switcher by user: %s",
					set.NetworkSwitchMode)
			}

		})
		err := r.Run()
		if err != nil {
			log.Panicf("failed to start server: %s", err)
		} else {
			log.Infof("server started")
		}
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
		//wg.Done()
	}()
	// переключатель режима сети
	go func() {
		for {
			switch <-networkModeChan {
			case "auto":
				log.Info("user switched to auto")
				err := set.NetworkAutoSwitch(rttCurrent, packetLossCurrent)
				if err != nil {
					fmt.Println("autoSwitch err: ", err)
				}
			case "reserve":
				log.Info("user switched to reserve")
				err := set.IpTablesSwitchReserve()
				if err != nil {
					fmt.Println("switch to reserve iptables mode err: ", err)
				}
			case "main":
				log.Info("user switched to main")
				err := set.IpTablesSwitchMain()
				if err != nil {
					fmt.Println("switch to reserve iptables mode err: ", err)
				}
			}
		}
	}()

	wg.Wait()

}

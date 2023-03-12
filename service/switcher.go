package service

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"log"
	"net"
	"net/http"
	"networkSwitcher/domain"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

func StartService() error {
	logErr := InitLogToFile()
	if logErr != nil {
		return fmt.Errorf("while creating config file: %w", logErr)
	}
	PingToSwitch := make(chan struct{})
	var set domain.MetricsCount
	err := set.IPTablesSetupInteface()
	if err != nil {
		fmt.Println(err)
	}
	validate := validator.New()
	// дефолтное значение параметров запуска утилиты
	set.RttSettings = 100          // предел задержки по сети
	set.PacketLossSettings = 60    // предел потери пакетов
	set.PingerCount = 10           // сколько пакетов надо плюнуть
	set.PingerInterval = 20        // за какое время это пакеты на выплюнуть
	set.NetworkSwitchMode = "auto" // автоматический режим переключения сети по умолчанию
	wg := sync.WaitGroup{}
	wg.Add(4)
	r := gin.Default()
	if err := Endpoints(r, &wg, validate, &set); err != nil {
		return err
	}
	if err := NetworkScan(PingToSwitch, &set); err != nil {
		return err
	}
	if err := Switch(PingToSwitch, &set); err != nil {
		return nil
	}
	wg.Wait()

	return nil
}

func Endpoints(r *gin.Engine, wg *sync.WaitGroup, validate *validator.Validate,
	set *domain.MetricsCount) error {
	go func() {
		// роут для получения пользователем информации о системе и насройках
		r.GET("/get_info", func(c *gin.Context) {
			c.JSON(http.StatusOK, set)
		})
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

			} else {

				set.NetworkSwitchMode = networkSwitchMode.NetworkSwitchMode

				c.IndentedJSON(http.StatusAccepted,
					"network mode: "+set.NetworkSwitchMode)
			}
		})
		err := r.Run()
		if err != nil {
			log.Panicf("failed to start server: %s", err)
		} else {
		}
		wg.Done()
	}()
	return nil
}
func NetworkScan(PingToSwitch chan struct{}, set *domain.MetricsCount) error {
	go func() {
		for {
			_, err := net.DialTimeout("tcp", "google.com:80",
				time.Second*2)
			if err != nil {
				time.Sleep(time.Millisecond * 50)
				continue
			}
			ping, err := exec.Command("ping", "-i 0.2",
				"-c 10", "8.8.8.8").Output()
			if err != nil {
				log.Println("while pinging: ", err)
			}
			stringPing := string(ping)
			packetLoss := strings.Split(stringPing, "\n")
			rttRow := packetLoss[len(packetLoss)-2]
			packetLossRow := packetLoss[len(packetLoss)-3]
			splittedPacketLossRow := strings.Split(packetLossRow, ",")
			finalPacketLoss, lossErr := strconv.ParseFloat(string(
				splittedPacketLossRow[2][1]), 64)
			if err != nil {
				log.Println(lossErr)
			}

			splittedRttRow := strings.Split(rttRow, "/")
			parseRtt := splittedRttRow[3]
			tt := strings.Split(parseRtt, " ")
			finalRtt, err := strconv.ParseFloat(tt[2], 64)
			if err != nil {
				log.Println(err)
			}
			set.Rtt = finalRtt
			set.PacketLoss = finalPacketLoss * 10
			PingToSwitch <- struct{}{}
		}
	}()
	return nil
}

func Switch(PingToSwitch chan struct{}, set *domain.MetricsCount) error {
	go func() {
		auto := false
		mainn := false
		reserve := false
		for {
			if set.NetworkSwitchMode == "auto" && !auto {

				if err := set.AutoNetwork(PingToSwitch); err != nil {
					log.Println(err)
				}

				log.Println("switched to auto")
				auto = true
				mainn = false
				reserve = false
			}
			if set.NetworkSwitchMode == "reserve" && !reserve {
				if err := set.IpTablesSwitchReserve(); err != nil {
					log.Println(err)
				}
				auto = false
				mainn = false
				reserve = true
				for set.NetworkSwitchMode == "reserve" {
					<-PingToSwitch
				}
			}
			if set.NetworkSwitchMode == "main" && !mainn {
				if err := set.IpTablesSwitchMain(); err != nil {
					log.Println(err)
				}
				auto = false
				reserve = false
				mainn = true
				for set.NetworkSwitchMode == "main" {
					<-PingToSwitch
				}
			}
			<-PingToSwitch
		}
	}()
	return nil
}

//
//go func() {
//	auto := false
//	mainn := false
//	reserve := false
//	for {
//		if set.NetworkSwitchMode == "auto" && !auto {
//
//			set.AutoNetwork(PingToSwitch)
//
//			fmt.Println("switched to auto")
//			auto = true
//			mainn = false
//			reserve = false
//		}
//		if set.NetworkSwitchMode == "reserve" && !reserve {
//			set.IpTablesSwitchReserve()
//			auto = false
//			mainn = false
//			reserve = true
//			for set.NetworkSwitchMode == "reserve" {
//				<-PingToSwitch
//			}
//		}
//		if set.NetworkSwitchMode == "main" && !mainn {
//			set.IpTablesSwitchMain()
//			auto = false
//			reserve = false
//			mainn = true
//			for set.NetworkSwitchMode == "main" {
//				<-PingToSwitch
//			}
//		}
//		<-PingToSwitch
//	}
//}()
//
//wg.Wait()

func InitLogToFile() error {
	logFile, createErr := os.OpenFile("NSlogfile.txt", os.O_RDWR|os.O_CREATE|
		os.O_CREATE, 0666)
	if createErr != nil {
		log.Println("cannot create lofFile: ", createErr)
	}
	log.SetOutput(logFile)
	defer logFile.Close()

	return nil
}

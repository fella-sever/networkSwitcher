package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MetricsCount struct {
	RttSettings         float64 `json:"rtt_settings"`                 // настройки ртт которые задает пользователь (в милисекундах)
	PacketLossSettings  float64 `json:"packet_loss_settings_percent"` // настройки потери пакетов, которые задает пользователь (в пакетах)
	Rtt                 float64 `json:"rtt_ms"`                       // реальный показатель ртт
	PacketLoss          float64 `json:"packet_loss_percent"`          // реальный показатель потерянных пакетов
	AliveMainNetwork    bool    `json:"alive_main_network"`           // состояние основного сетевого интерфейса
	AliveReserveNetwork bool    `json:"alive_reserve_network"`        // состояние резервного сетевого интерфейса
	PingerCount         int     `json:"pinger_count"`                 // настройки количества пакетов при тестировании сети (пользователь)
	PingerInterval      int64   `json:"pinger_interval_ms"`           // настройки интервалов пинга (пользователь)
	NetworkSwitchMode   string  `json:"network_switch_mode"`          // настройки режима переключения сети
}

type MetricsUserSetDto struct {
	RttSettings    float64 `json:"rtt_settings_ms" validate:"required"`
	PacketLoss     float64 `json:"packet_loss_percent" validate:"required"`
	PingerCount    int     `json:"pinger_count"`
	PingerInterval int64   `json:"pinger_interval_ms" validate:"numeric"`
}
type NetworkSwitchSettingsUserSetDTO struct {
	NetworkSwitchMode string `json:"network_switch_mode" validate:"eq=main|eq=auto|eq=reserve,required"`
}

func main() {
	PingToSwitch := make(chan struct{})
	var set MetricsCount
	validate := validator.New()
	// дефолтное значение параметров запуска утилиты
	set.RttSettings = 100          // предел задержки по сети
	set.PacketLossSettings = 60    // предел потери пакетов
	set.PingerCount = 10           // сколько пакетов надо плюнуть
	set.PingerInterval = 20        // за какое время это пакеты на выплюнуть
	set.NetworkSwitchMode = "auto" // автоматический режим переключения сети по умолчанию
	wg := sync.WaitGroup{}
	wg.Add(4)
	// запуск сервера
	r := gin.Default()
	go func() {
		// роут для получения пользователем информации о системе и насройках
		r.GET("/get_info", func(c *gin.Context) {
			c.JSON(http.StatusOK, set)
		})
		r.POST("/set_threshold", func(c *gin.Context) {
			var newSettings MetricsUserSetDto
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

			var networkSwitchMode NetworkSwitchSettingsUserSetDTO
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

	//запуск сканирования состояния сети
	go func() {
		for {
			_, err := net.DialTimeout("tcp", "google.com:80", time.Second*2)
			if err != nil {
				time.Sleep(time.Millisecond * 50)
				continue
			}
			ping, err := exec.Command("ping", "-i 0.2",
				"-c 10", "8.8.8.8").Output()
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
			PingToSwitch <- struct{}{}
		}
	}()

	go func() {
		auto := false
		mainn := false
		reserve := false
		for {
			if set.NetworkSwitchMode == "auto" && !auto {

				set.AutoNetwork(PingToSwitch)

				fmt.Println("switched to auto")
				auto = true
				mainn = false
				reserve = false
			}
			if set.NetworkSwitchMode == "reserve" && !reserve {
				set.IpTablesSwitchReserve()
				auto = false
				mainn = false
				reserve = true
				for set.NetworkSwitchMode == "reserve" {
					<-PingToSwitch
				}
			}
			if set.NetworkSwitchMode == "main" && !mainn {
				set.IpTablesSwitchMain()
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

	wg.Wait()
}

func (m *MetricsCount) AutoNetwork(ch chan struct{}) error {
	switchCount := 1
	switchCountPacket := 1
	IsMain := false
	IsReserve := false
	for m.NetworkSwitchMode == "auto" {
		<-ch
		fmt.Println("inner auto")

		if m.Rtt > m.RttSettings && switchCount == 0 {
			switchCount++
			if !IsReserve {
				if err := m.IpTablesSwitchReserve(); err != nil {
					return fmt.Errorf("auto switch err: %w", err)
				}
				IsReserve = true
				IsMain = false
			}
		} else if m.Rtt < m.RttSettings && switchCount == 1 {
			switchCount--
			if !IsMain {
				if err := m.IpTablesSwitchMain(); err != nil {
					return fmt.Errorf("auto switch err: %w", err)
				}
				IsMain = true
				IsReserve = false
			}
		}
		if m.PacketLoss > m.PacketLossSettings && switchCountPacket == 0 {
			switchCountPacket++
			if !IsReserve {
				if err := m.IpTablesSwitchReserve(); err != nil {
					return fmt.Errorf("auto switch err: %w", err)
				}
				IsReserve = true
				IsMain = false
			}
		} else if m.PacketLoss <= m.PacketLossSettings && switchCountPacket == 1 {
			switchCountPacket--
			if !IsMain {
				if err := m.IpTablesSwitchMain(); err != nil {
					return fmt.Errorf("auto switch err: %w", err)
				}
				IsMain = true
				IsReserve = false
			}
		}

	}

	return nil
}

// IpTablesSwitchMain
// запуск заранее подготовленного скрипта для очистки таблиц маршрутизации и
// загрузки новых под основную сеть
func (m *MetricsCount) IpTablesSwitchMain() error {
	log.Println("main")
	_, err := exec.Command("ifconfig", "main", "up").Output()
	if err != nil {
		return err
	}
	_, err = exec.Command("ifconfig", "reserve", "down").Output()
	if err != nil {
		return err
	}

	return nil
}

// IpTablesSwitchReserve
// запуск заранее подготовленного скрипта для очистки таблиц маршрутизации и
// загрузки новых под резервную сеть
func (m *MetricsCount) IpTablesSwitchReserve() error {
	_, err := exec.Command("ifconfig", "reserve", "up").Output()
	if err != nil {
		return err
	}
	_, err = exec.Command("ifconfig", "main", "down").Output()
	if err != nil {
		return err
	}
	log.Println("reserve")
	return nil
}

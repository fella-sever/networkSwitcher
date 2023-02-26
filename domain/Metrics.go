package domain

import (
	"fmt"
	"net"
	"time"
)

type MetricsCount struct {
	RttSettings float64 `json:"rtt_settings"` // настройки ртт,
	// которые задает пользователь (в милисекундах)
	PacketLossSettings  float64 `json:"packet_loss_settings_percent"` // настройки потери пакетов, которые задает пользователь (в пакетах)
	Rtt                 float64 `json:"rtt_ms"`                       // реальный показатель ртт
	PacketLoss          float64 `json:"packet_loss_percent"`          // реальный показатель потерянных пакетов
	AliveMainNetwork    bool    `json:"alive_main_network"`           // состояние основного сетевого интерфейса
	AliveReserveNetwork bool    `json:"alive_reserve_network"`        // состояние резервного сетевого интерфейса
	PingerCount         int     `json:"pinger_count"`                 // настройки количества пакетов при тестировании сети (пользователь)
	PingerInterval      int64   `json:"pinger_interval_ms"`           // настройки интервалов пинга (пользователь)
	NetworkSwitchMode   string  `json:"network_switch_mode"`          // настройки режима переключения сети
}

type MetricsSetDto struct {
	RttSettings    float64 `json:"rtt_settings_ms" validate:"required"`
	PacketLoss     float64 `json:"packet_loss_percent" validate:"required"`
	PingerCount    int     `json:"pinger_count"`
	PingerInterval int64   `json:"pinger_interval_ms" validate:"numeric"`
}
type NetworkSwitchSettingsDTO struct {
	NetworkSwitchMode string `json:"network_switch_mode" validate:"eq=main|eq=auto|eq=reserve,required"`
}

func (m *MetricsCount) CheckPacketLossRttThreshold(rtt chan time.Duration,
	packetLoss chan float64, swCount *int) error {

	return nil
}

func (m *MetricsCount) CheckInterfaceIsAlive(interfaceName string) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("check interface is alive: %w", err)
	}
	for _, val := range interfaces {
		if val.Name == interfaceName && interfaceName == "main" {
			m.AliveMainNetwork = true
			break
		}

		if val.Name == interfaceName && interfaceName == "reserve" {
			m.AliveReserveNetwork = true
			break
		}
	}
	return nil
}

func IpTablesSwitchMain() error {
	fmt.Println("switched to main")
	return nil
}

func IpTablesSwitchReseve() error {
	fmt.Println("switched to reserve")
	return nil
}

func (m *MetricsCount) NetworkAutoSwitch(rttCount chan float64,
	packetLossCount chan float64) error {
	switchCount := 1
	switchCountPacket := 1
	IsMain := false
	IsReserve := false

	for {
		rttCountInner := <-rttCount
		packetLOssInner := <-packetLossCount
		if m.NetworkSwitchMode != "auto" {
			break
		}
		if rttCountInner > m.RttSettings && switchCount == 0 {
			switchCount++
			if !IsReserve {
				if err := IpTablesSwitchReseve(); err != nil {
					return fmt.Errorf("auto switch err: %w", err)
				}
				IsReserve = true
				IsMain = false
			}
		} else if rttCountInner < m.RttSettings && switchCount == 1 {
			switchCount--
			if !IsMain {
				if err := IpTablesSwitchMain(); err != nil {
					return fmt.Errorf("auto switch err: %w", err)
				}
				IsMain = true
				IsReserve = false
			}
		}
		if packetLOssInner > m.PacketLossSettings && switchCountPacket == 0 {
			switchCountPacket++
			if !IsReserve {
				if err := IpTablesSwitchReseve(); err != nil {
					return fmt.Errorf("auto switch err: %w", err)
				}
				IsReserve = true
				IsMain = false
			}
		} else if packetLOssInner <= m.PacketLossSettings && switchCountPacket == 1 {
			switchCountPacket--
			if !IsMain {
				if err := IpTablesSwitchMain(); err != nil {
					return fmt.Errorf("auto switch err: %w", err)
				}
				IsMain = true
				IsReserve = false
			}
		}
	}
	return nil
}

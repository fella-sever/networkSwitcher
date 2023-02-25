package domain

import (
	"net"
	"sync"
	"time"
)

type MetricsCount struct {
	RttSettings float64 `json:"rtt_settings"` // настройки ртт,
	// которые задает пользователь (в милисекундах)
	PacketLossSettings  float64 `json:"packet_loss_settings_percent"` // настройки потери пакетов, которые задает пользователь (в пакетах)
	Rtt                 float64 `json:"rtt_ms"`                       // реальный показатель ртт
	PacketLoss          int     `json:"packet_loss_percent"`          // реальный показатель потерянных пакетов
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

// PacketLossCount
// функция для подсчета
func (m *MetricsCount) NetworkCheckService(rtt chan time.Duration, packetLoss chan float64, mu sync.RWMutex) error {

	return nil
}

func (m *MetricsCount) CheckPacketLossRttThreshold(rtt chan time.Duration,
	packetLoss chan float64, swCount *int) error {

	return nil
}

func (m *MetricsCount) CheckInterfaceIsAlive(interfaceName string) (bool, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false, err
	}
	for _, val := range interfaces {
		if interfaceName == val.Name {
			return true, nil
		}
	}
	return false, nil
}

func IpTablesSwitchMain() error {
	return nil
}

func IpTablesSwitchReseve() error {
	return nil
}

func (m *MetricsCount) SetNetworkSwitchMode() error {

	return nil
}

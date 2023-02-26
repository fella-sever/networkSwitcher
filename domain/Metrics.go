package domain

import (
	"fmt"
	"net"
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

type MetricsUserSetDto struct {
	RttSettings    float64 `json:"rtt_settings_ms" validate:"required"`
	PacketLoss     float64 `json:"packet_loss_percent" validate:"required"`
	PingerCount    int     `json:"pinger_count"`
	PingerInterval int64   `json:"pinger_interval_ms" validate:"numeric"`
}
type NetworkSwitchSettingsUserSetDTO struct {
	NetworkSwitchMode string `json:"network_switch_mode" validate:"eq=main|eq=auto|eq=reserve,required"`
}

// CheckInterfaceIsAlive
// проверка наличия сетевого интерфейса в системе. (для наглядности я буду
// переименовывать интерфейсы в main - основная сеть и reserve - сеть свистка)
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

// IpTablesSwitchMain
// запуск заранее подготовленного скрипта для очистки таблиц маршрутизации и
// загрузки новых под основную сеть
func IpTablesSwitchMain() error {
	// TODO implement me!
	fmt.Println("switched to main")
	return nil
}

// IpTablesSwitchReserve
// запуск заранее подготовленного скрипта для очистки таблиц маршрутизации и
// загрузки новых под резервную сеть
func IpTablesSwitchReserve() error {
	// TODO implement me!
	fmt.Println("switched to reserve")
	return nil
}

// NetworkAutoSwitch
// метод для автоматического переключения сети в зависимости от метрик ртт и потери
// пакетов. Принимает на вход метрики через два канала синхронизатора, считывание
// из которых, во-первых, служит как тормоз для суперцикла,
// во-вторых - передает внутрь информацию для работы алгоритма
func (m *MetricsCount) NetworkAutoSwitch(rttCount chan float64,
	packetLossCount chan float64) error {
	// семафоры-семафорчики для того, чтобы у нас не было кучи переключений, а так же
	// двойного переключения сети в том случае, если у нас меняются сразу две метрики
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
				if err := IpTablesSwitchReserve(); err != nil {
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
				if err := IpTablesSwitchReserve(); err != nil {
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

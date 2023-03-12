package domain

import (
	"fmt"
	"log"
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

func (m *MetricsCount) AutoNetwork(ch chan struct{}) error {
	switchCount := 1
	switchCountPacket := 1
	IsMain := false
	IsReserve := false
	for m.NetworkSwitchMode == "auto" {
		<-ch

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
	////iptables --table nat --append POSTROUTING --out-interface eth0 -j MASQUERADE
	//_, mainResErr := exec.Command("iptables",
	//	"--table", "nat", "--append", "POSTROUTING",
	//	"--out-interface", "eth0", "-j", "MASQUERADE").
	//	Output()
	//if mainResErr != nil {
	//	log.Println("while switching to main network: ", mainResErr)
	//}
	//_, resNetErr := exec.Command("iptables",
	//	"--table", "nat", "--delete", "POSTROUTING",
	//	"--out-interface", "wan0", "-j", "MASQUERADE").
	//	Output()
	//if resNetErr != nil {
	//	log.Println("while switching to main network: ", mainResErr)
	//}

	return nil
}

// IpTablesSwitchReserve
// запуск заранее подготовленного скрипта для очистки таблиц маршрутизации и
// загрузки новых под резервную сеть
func (m *MetricsCount) IpTablesSwitchReserve() error {
	log.Println("reserve")
	//_, resNetErr := exec.Command("iptables",
	//	"--table", "nat", "--append", "POSTROUTING",
	//	"--out-interface", "wan0", "-j", "MASQUERADE").
	//	Output()
	//if resNetErr != nil {
	//	log.Println("reserve switch to reserve:", resNetErr)
	//}
	//_, mainNetErr := exec.Command("iptables",
	//	"--table", "nat", "--delete", "POSTROUTING",
	//	"--out-interface", "eth0", "-j", "MASQUERADE").
	//	Output()
	//if mainNetErr != nil {
	//	return fmt.Errorf("while switching to reserve network: %w", mainNetErr)
	//}
	return nil
}

func (m *MetricsCount) IPTablesSetupInteface() error {
	//_, err := exec.Command("iptables",
	//	"--append", "FORWARD", "--in-interface", "eth1", "-j", "ACCEPT").Output()
	//if err != nil {
	//	return fmt.Errorf("while setup lan0 interface %w", err)
	//}
	return nil
}

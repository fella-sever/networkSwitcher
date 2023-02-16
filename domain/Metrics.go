package domain

import (
	"fmt"
	"github.com/go-ping/ping"
	"net"
	"time"
)

type MetricsCount struct {
	RttSettings         string  `json:"rtt_settings"`
	PacketLossSettings  float64 `json:"packet_loss_settings"`
	Rtt                 string  `json:"rtt"`
	PacketLoss          float64 `json:"packet_loss"`
	AliveMainNetwork    bool    `json:"alive_main_network"`
	AliveReserveNetwork bool    `json:"alive_reserve_network"`
}

type MetricsSetDto struct {
	RttSettings string  `json:"rtt_settings" validate:"required, numeric"`
	PacketLoss  float64 `json:"packet_loss" validate:"required"`
}

// PacketLossCount
// функция для подсчета
func (m *MetricsCount) PacketLossRttCount() error {
	pinger, err := ping.NewPinger("www.google.com")
	if err != nil {
		return err
	}
	pinger.Count = 10
	pinger.Interval = time.Millisecond * 20
	err = pinger.Resolve()
	err = pinger.Run()
	if err != nil {
		return nil
	}
	m.Rtt = pinger.Statistics().AvgRtt.String()
	m.PacketLoss = pinger.Statistics().PacketLoss
	return nil
}

func (m *MetricsCount) CheckPacketLossRttThreshold() error {
	swCount := 1
	for {
		if m.Rtt > m.RttSettings && m.PacketLoss > m.PacketLossSettings && swCount == 0 {
			fmt.Println("switch to reserve network")
			swCount++
		} else if m.Rtt < m.RttSettings && swCount == 1 {
			fmt.Println("switch to main network")
			swCount--
		}
	}
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

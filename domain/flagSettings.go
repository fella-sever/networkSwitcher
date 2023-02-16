package domain

// flags defines parameters to network switching from main channel to reserve and another

type SettingsFlags struct {
	Rtt        int // in microseconds
	PacketLoss int // in packet per minute
}

package state

type Session struct {
	User string
	PartitionID string
	IsActive bool
}

var CurrentSession Session
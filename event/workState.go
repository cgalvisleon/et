package event

type WorkStatus int

const (
	WorkStatusPending WorkStatus = iota
	WorkStatusAccepted
	WorkStatusProcessing
	WorkStatusCompleted
	WorkStatusFailed
)

/**
* String
* @return string
**/
func (s WorkStatus) String() string {
	switch s {
	case WorkStatusPending:
		return "Pending"
	case WorkStatusAccepted:
		return "Accepted"
	case WorkStatusProcessing:
		return "Processing"
	case WorkStatusCompleted:
		return "Completed"
	case WorkStatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

/**
* ToWorkStatus
* @param int n
* @return WorkStatus
**/
func ToWorkStatus(n int) WorkStatus {
	switch n {
	case 0:
		return WorkStatusPending
	case 1:
		return WorkStatusAccepted
	case 2:
		return WorkStatusProcessing
	case 3:
		return WorkStatusCompleted
	case 4:
		return WorkStatusFailed
	default:
		return WorkStatusPending
	}
}

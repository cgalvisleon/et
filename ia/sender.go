package ia

import "github.com/cgalvisleon/et/et"

type Sender interface {
	SendTextMessage(to string, content string) (et.Item, error)
}

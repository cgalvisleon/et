package service

func init() {
	senders = make(map[string]*Send)
	storages = make(map[string]*LocalStorage)
}

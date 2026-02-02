package stream

var stopStreamChan = make(chan struct{}, 1)

// Остановить текущий поток (без блокировки)
func StopCurrentStream() {
	select {
	case stopStreamChan <- struct{}{}:
	default:
	}
}

// Вернуть канал для StreamRadio
func StopChan() <-chan struct{} {
	return stopStreamChan
}

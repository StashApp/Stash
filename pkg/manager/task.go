package manager

import "sync"

type Task interface {
	Start(wg *sync.WaitGroup)
	GetDescription() string
}

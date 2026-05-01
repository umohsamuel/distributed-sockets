package service

import "github.com/umohsamuel/distributed-sockets/internal/adapter"

type ServiceDependencies struct {
	Adapter *adapter.Adapters
}

type Services struct {
	Adapter *adapter.Adapters
}

func NewService(deps ServiceDependencies) *Services {
	return &Services{
		Adapter: deps.Adapter,
	}
}

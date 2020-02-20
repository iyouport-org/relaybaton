package config

type Config interface {
	Init() error
	validate() error
}

package main

import (

)

type Node interface {
	Join(network interface{}) error
	Quit(network interface{}) error
	Broadcast(message interface{})
	BlockGen() interface{}
	Query(request interface{}) interface{}
}
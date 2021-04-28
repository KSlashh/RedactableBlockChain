package main

import (
	ch "github.com/RedactableBlockChain/chameleon"
	"github.com/RedactableBlockChain/data"
	"github.com/RedactableBlockChain/path"
)

func main() {
	var p,q,g,hk,tk []byte
	ch.ParameterGen(128, &p, &q, &g)
	ch.Keygen(128, p, q, g, &hk, &tk)
	config := &data.GolbalParameter{
		0,
		128,
		p,
		q,
		g,
		hk,
		tk,
	}
	data.Write(config, path.GetConfigPath())
}
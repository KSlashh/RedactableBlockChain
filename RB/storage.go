package main

import (
	"encoding/json"
	"os"
)

func (t *BasicTx) Write(path string) error {
	fw,err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer fw.Close()

	encoder := json.NewEncoder(fw)
	err = encoder.Encode(t)
	if err != nil {
		return err
	}

	return nil
}

func Read(path string) (*BasicTx,error) {
	t := &BasicTx{}
	fr,err := os.Open(path)
	if err != nil {
		return t,err
	}
	defer fr.Close()

	decoder := json.NewDecoder(fr)
	err = decoder.Decode(&t)
	if err != nil {
		return t,err
	}
	return t,nil
}

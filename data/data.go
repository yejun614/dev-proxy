package data

import (
	"encoding/json"
	"os"
)

type Data[T any] struct {
	Filepath string `json:"filepath" xml:"filepath" form:"filepath"`
	Data     T      `json:"data" xml:"data" form:"data"`
}

func New[T any](filepath string, sample T) *Data[T] {
	// create data
	data := new(Data[T])
	// set
	data.Filepath = filepath
	// open data
	err := data.Open()
	if err != nil {
		data.Data = sample
		data.Save() // create new file
	}
	// return data
	return data
}

func (data *Data[T]) Open() error {
	f, err := os.Open(data.Filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	decoder.Decode(&data.Data)
	return nil
}

func (data *Data[T]) Save() error {
	f, err := os.Create(data.Filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ") // pretty json
	encoder.Encode(data.Data)
	return nil
}

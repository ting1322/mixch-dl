package inter

import (
	"io"
	"log"
	"os"
)

type IFs interface {
	Save(filename string, data []byte) error
	Exist(filename string) bool
	Create(filename string) (io.WriteCloser, error)
	Delete(filename string)
}

type Fs struct {
}

func (this Fs) Save(filename string, data []byte) error {
	file, err := this.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	file.Write(data)
	return nil
}

func (this Fs) Exist(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return true
}

func (this Fs) Create(filename string) (io.WriteCloser, error) {
	log.Println("CREATE:", filename)
	return os.Create(filename)
}

func (this Fs) Delete(filename string) {
	log.Println("DELETE:", filename)
	os.Remove(filename)
}

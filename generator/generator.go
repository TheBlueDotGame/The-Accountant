package generator

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"

	"github.com/bartossh/Computantis/emulator"
)

// GenerateToFile generates data to file in json format.
func GenerateToFile(filePath string, count, vMin, vMax, maMin, maMax int) error {
	if vMin >= vMax || maMin >= maMax || count == 0 {
		return errors.New("wrong parameter")
	}

	data := make([]emulator.Measurement, 0, count)
	for i := 0; i < count; i++ {
		volts := rand.Intn(vMax-vMin) + vMin
		mamps := rand.Intn(maMax-maMin) + maMin
		m := emulator.Measurement{
			Volts: volts,
			Mamps: mamps,
			Power: volts * mamps / 1000,
		}

		data = append(data, m)
	}

	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, file, 0644)
}

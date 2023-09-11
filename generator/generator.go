package generator

import (
	"encoding/json"
	"errors"
	"math/rand"
	"os"

	"github.com/bartossh/Computantis/emulator"
)

// ToJSONFile generates data to file in json format.
func ToJSONFile(filePath string, count, vMin, vMax, maMin, maMax int64) error {
	if vMin >= vMax || maMin >= maMax || count == 0 {
		return errors.New("wrong parameter")
	}

	data := make([]emulator.Measurement, 0, count)
	var i int64 = 0
	for ; i < count; i++ {
		volts := rand.Int63n(vMax-vMin) + vMin
		mamps := rand.Int63n(maMax-maMin) + maMin
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

	return os.WriteFile(filePath, file, 0644)
}

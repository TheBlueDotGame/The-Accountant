package logo

import (
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func Display() {
	s, _ := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("Compu", pterm.FgCyan.ToStyle()),
		putils.LettersFromStringWithStyle("tantis", pterm.FgLightMagenta.ToStyle())).Srender()
	pterm.DefaultCenter.Println(s)
	pterm.DefaultCenter.WithCenterEachLineSeparately().
		Println("This software belongs to\nThe Computantis Project\n(C) 2023.")
}

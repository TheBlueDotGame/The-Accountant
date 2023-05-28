package logo

import (
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func Display() {
	s, _ := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("C", pterm.FgCyan.ToStyle()),
		putils.LettersFromStringWithStyle("omputantis", pterm.FgLightMagenta.ToStyle())).Srender()
	pterm.DefaultCenter.Println(s)
	pterm.DefaultCenter.WithCenterEachLineSeparately().
		Println("This software belongs to\nComputantis Project\n and was build with passion.")
}

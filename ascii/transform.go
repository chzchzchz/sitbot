package ascii

type Transform func(*ASCII)

func PaletteFg(a *ASCII) {
	for r, row := range a.Cells {
		for c, col := range row {
			a.Cells[r][c].Value = rune(index2byte(lookupIndex(col.Foreground)))
			a.Cells[r][c].Background = a.Cells[r][c].Foreground
			a.Cells[r][c].Foreground = nil
		}
	}
}

func PaletteBg(a *ASCII) {
	for r, row := range a.Cells {
		for c, col := range row {
			a.Cells[r][c].Value = rune(index2byte(lookupIndex(col.Background)))
			a.Cells[r][c].Foreground = nil
		}
	}
}

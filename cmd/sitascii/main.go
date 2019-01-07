package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/andlabs/ui"
	"github.com/chzchzchz/sitbot/ascii"
)

const winH = 600
const winW = 800

var fontButton *ui.FontButton
var attrstr = ui.NewAttributedString("")
var selectColor *SelectColor

func uiFunc() {
	window := ui.NewWindow("SITASCII STUDIO", winW, winH, true)
	window.SetMargined(true)
	board := NewBoard()

	loadAscii := func(f string) error {
		dat, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}
		aa, err := ascii.NewASCII(dat)
		if err != nil {
			return err
		}
		board.SetASCII(aa)
		fmt.Println("loaded ascii", f)
		return nil
	}

	if len(os.Args) > 1 {
		loadAscii(os.Args[1])
	}

	loadButton := ui.NewButton("Load")
	loadButton.OnClicked(func(*ui.Button) {
		loadAscii(ui.OpenFile(window))
	})
	saveButton := ui.NewButton("Save")
	saveButton.OnClicked(func(*ui.Button) {
		s := ui.SaveFile(window)
		if err := ioutil.WriteFile(s, board.ASCII().Bytes(), 0644); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("saved to", s)
	})
	newButton := ui.NewButton("New")
	newButton.OnClicked(func(*ui.Button) {
		a, _ := ascii.NewASCII(nil)
		board.SetASCII(a)
	})

	entryHBox := ui.NewHorizontalBox()
	entryLabel := ui.NewLabel("Brush ")
	entryChar := ui.NewEntry()
	entryChar.OnChanged(func(e *ui.Entry) {
		t := e.Text()
		board.Brush = t
	})
	entryHBox.Append(entryLabel, false)
	entryHBox.Append(entryChar, true)

	fontButton = ui.NewFontButton()
	fontButton.OnChanged(func(*ui.FontButton) {
		uiFont = fontButton.Font()
		board.QueueRedrawAll()
	})

	selectColor = NewSelectColor(ascii.NewPaletteMIRC())

	vbox := ui.NewVerticalBox()
	for _, c := range []ui.Control{
		newButton,
		loadButton,
		saveButton,
		entryHBox,
		fontButton,
	} {
		vbox.Append(c, false)
	}
	vbox.Append(selectColor, true)

	hbox := ui.NewHorizontalBox()
	hbox.SetPadded(true)
	hbox.Append(vbox, false)
	hbox.Append(board, true)
	window.SetChild(hbox)
	window.OnClosing(func(*ui.Window) bool {
		ui.Quit()
		return true
	})
	window.Show()
}

func GetSelectColor() ascii.ColorPair {
	return selectColor.ColorPair
}

func SetSelectColor(c ascii.ColorPair) {
	selectColor.ColorPair = c
	selectColor.QueueRedrawAll()
}

func main() {
	err := ui.Main(uiFunc)
	if err != nil {
		panic(err)
	}
}

//go:build tinygo

package main

import (
	"machine"
	"time"

	font "github.com/Nondzu/ssd1306_font"
	"tinygo.org/x/drivers/ssd1306"
)

// The panel on this board is the same 128x32 SSD1306 the earlier
// firmware drove: I2C0 on GPIO21 (SDA) and GPIO22 (SCL).
const (
	mantyWord = "MANTY "
	glyphW    = 7
	glyphH    = 10
	panelW    = 128
	panelRows = 3
)

// newDisplay brings up the OLED and a 7x10 font. The returned font shares
// the panel's frame buffer, so drawManty can clear that buffer and draw
// into it.
func newDisplay() (*ssd1306.Device, *font.Display) {
	machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
		SDA:       machine.GPIO21,
		SCL:       machine.GPIO22,
	})

	screen := ssd1306.NewI2C(machine.I2C0)
	screen.Configure(ssd1306.Config{
		Address: ssd1306.Address_128_32,
		Width:   panelW,
		Height:  32,
	})
	screen.ClearDisplay()

	text := font.NewDisplay(*screen)
	text.Configure(font.Config{FontType: font.FONT_7x10})
	return screen, &text
}

// runManty keeps writing MANTY across the panel. Each frame shifts one
// character, and each row starts two characters later than the row above,
// so the screen is a moving repeat instead of the old two static lines.
func runManty(screen *ssd1306.Device, text *font.Display) {
	step := 0
	for {
		drawManty(screen, text, step)
		step = (step + 1) % len(mantyWord)
		time.Sleep(400 * time.Millisecond)
	}
}

// drawManty fills the 128x32 buffer with MANTY, starting at character step.
func drawManty(screen *ssd1306.Device, text *font.Display, step int) {
	screen.ClearBuffer()
	n := len(mantyWord)
	for row := 0; row < panelRows; row++ {
		text.YPos = int16(row * glyphH)
		text.XPos = 0
		idx := (step + row*2) % n
		for text.XPos <= panelW-glyphW {
			text.PrintChar(mantyWord[idx])
			text.XPos += glyphW
			idx++
			if idx == n {
				idx = 0
			}
		}
	}
	_ = screen.Display()
}

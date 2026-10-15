package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/mdp/qrterminal/v3"
)

// MaxQRBytes is the most text that fits in a scannable terminal QR code.
const MaxQRBytes = 1200

// QRString renders text as a half-block QR code.
func QRString(text string) (string, error) {
	if len(text) > MaxQRBytes {
		return "", fmt.Errorf("too long for a QR code (%d bytes, max %d)", len(text), MaxQRBytes)
	}
	var sb strings.Builder
	qrterminal.GenerateHalfBlock(text, qrterminal.L, &sb)
	return sb.String(), nil
}

// PrintQR writes a QR code for text to w.
func PrintQR(w io.Writer, text string) error {
	s, err := QRString(text)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, s)
	return err
}

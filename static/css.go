//go:generate ../_output/deps/tailwindcss -i ./css/input.css -o ./css/output.css
package static

import (
	_ "embed"
	"io"
)

//go:embed css/output.css
var css []byte

func RenderStyle(w io.Writer) error {
	_, err := w.Write(css)
	return err
}

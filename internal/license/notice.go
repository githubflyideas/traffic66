package license

// The notice an unregistered installation puts on what it generates, so
// that it travels with the output rather than with the page that showed
// it.
//
// It is plain English rather than one of the translated strings: the pages
// are translated in the browser, and a file that has left the installation
// carries no language setting to follow.
const noticeText = "EVALUATION COPY - this traffic66 installation is not registered. " +
	"Figures are complete; the copy is not licensed for distribution."

// Notice is the line to put on anything this installation generates that
// can leave the browser. It is empty when the installation is registered,
// which is what callers test: an empty notice is added to nothing.
func (s State) Notice() string {
	if !s.Watermark() {
		return ""
	}
	return noticeText
}

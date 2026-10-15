package license

// The evaluation mark used to be drawn only by the browser, as a CSS
// background over the view. That marks what somebody looks at and nothing
// else: a CSV downloaded from the API, a findings list fetched by a script,
// a table copied as text and pasted into a document all came out clean, and
// turning the mark off was a matter of not loading the stylesheet. Anything
// generated here carries the notice instead, so it travels with the file
// rather than with the page that showed it.
//
// The wording is deliberately plain and in English. The pages are
// translated in the browser, but a file that has left the installation has
// no language setting to follow, and it is read by whoever received it.
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

package api

import (
	"net/http"
	"strings"

	"github.com/githubflyideas/traffic66/internal/license"
)

// noticeHeader carries the evaluation notice on every API response, so a
// script that fetches JSON sees it without the body having to carry it.
const noticeHeader = "X-Traffic66-Evaluation"

// notice is the evaluation line for anything this server generates, empty
// when the installation is registered. A server with no licence checker at
// all (the offline sandbox) is treated as unregistered, the same way
// allowed() treats it: the answer is never more permissive than a missing
// licence.
func (s *Server) notice() string {
	if s.License == nil {
		return license.Unregistered("").Notice()
	}
	return s.License.State().Notice()
}

// markHeader puts the notice on a response. It is called for every API
// route from one place rather than per handler, so a route added later
// carries it without anybody remembering to.
func (s *Server) markHeader(w http.ResponseWriter) {
	if n := s.notice(); n != "" {
		w.Header().Set(noticeHeader, n)
	}
}

// commentLines renders the notice as leading comment lines for a text
// format, empty when there is nothing to say.
func commentLines(notice, prefix string) string {
	if notice == "" {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(notice, "\n") {
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

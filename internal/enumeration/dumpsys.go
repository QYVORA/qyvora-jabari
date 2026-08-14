package enumeration

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// Field-level parsers for the "dumpsys package <pkg>" output. The output
// format is not a stable API, so parsing is deliberately tolerant: any line
// that does not match is skipped and the application entry keeps whatever
// fields were successfully extracted.
var (
	reVersionCode = regexp.MustCompile(`versionCode=(\d+)`)
	reVersionName = regexp.MustCompile(`versionName=(\S+)`)
	rePkgFlags    = regexp.MustCompile(`pkgFlags=\[([^\]]*)\]`)
	reUserId      = regexp.MustCompile(`userId=(\d+)`)
)

// parsePackageDump extracts the security-relevant fields an app inventory
// needs from a dumpsys package listing. Only fields that could not be parsed
// remain at their zero value.
func parsePackageDump(out, pkg string) models.Application {
	app := models.Application{PackageName: pkg}
	lines := strings.Split(out, "\n")

	inPermissions := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track entry into and exit from the requested-permissions section.
		// The section ends at the next top-level heading (a line without leading
		// whitespace) or at another "xxx:" subsection header such as
		// "install permissions:". Permission lines are always plain permission
		// names.
		switch {
		case strings.HasPrefix(trimmed, "requested permissions:"):
			inPermissions = true
			continue
		case inPermissions && strings.HasSuffix(trimmed, ":") && strings.Contains(trimmed, " "):
			inPermissions = false
		case inPermissions && trimmed != "" && line == trimmed:
			inPermissions = false
		case inPermissions && trimmed != "":
			app.Permissions = append(app.Permissions, trimmed)
			continue
		}

		if m := reVersionCode.FindStringSubmatch(line); m != nil {
			app.VersionCode = m[1]
		}
		if m := reVersionName.FindStringSubmatch(line); m != nil {
			app.VersionName = m[1]
		}
		if m := rePkgFlags.FindStringSubmatch(line); m != nil {
			flags := strings.ToUpper(m[1])
			if strings.Contains(flags, "DEBUGGABLE") {
				app.Debuggable = true
			}
			if strings.Contains(flags, "SYSTEM") {
				app.SystemApp = true
			}
		}
		if m := reUserId.FindStringSubmatch(line); m != nil {
			if uid, err := strconv.Atoi(m[1]); err == nil {
				app.Attributes = map[string]string{"uid": strconv.Itoa(uid)}
			}
		}
	}
	return app
}

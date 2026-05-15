// Package logx provides readable, optionally colored stderr logging for CLI use.
package logx

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
)

var (
	stampStyles *color.Color
	labelInfo   *color.Color
	labelWarn   *color.Color
	labelErr    *color.Color
	labelOK     *color.Color
	labelDebug  *color.Color
	emPkg       *color.Color
	emVer       *color.Color
	emHighlight *color.Color
	secTitle    *color.Color
	secLine     *color.Color
	stepArrow   *color.Color
	stepLabel   *color.Color
	idxMuted    *color.Color
	linkMuted   *color.Color
	debugOn     bool
)

func init() {
	color.Output = colorable.NewColorable(os.Stderr)

	if os.Getenv("NO_COLOR") != "" || os.Getenv("CI") != "" {
		color.NoColor = true
	}

	stampStyles = color.New(color.Faint)
	labelInfo = color.New(color.FgHiCyan)
	labelWarn = color.New(color.FgHiYellow)
	labelErr = color.New(color.FgHiRed)
	labelOK = color.New(color.FgHiGreen)
	labelDebug = color.New(color.FgHiMagenta)
	emPkg = color.New(color.FgCyan)
	emVer = color.New(color.FgYellow)
	emHighlight = color.New(color.Bold, color.FgWhite)
	secTitle = color.New(color.Bold, color.FgMagenta)
	secLine = color.New(color.Faint)
	stepArrow = color.New(color.FgGreen)
	stepLabel = color.New(color.Bold, color.FgWhite)
	idxMuted = color.New(color.Faint)
	linkMuted = color.New(color.Faint)
}

func stamp() string {
	ts := time.Now().Format("15:04:05")
	if color.NoColor {
		return fmt.Sprintf("[%s] ", ts)
	}
	return stampStyles.Sprintf("[%s] ", ts)
}

// DisableColor forces plain output (same effect as NO_COLOR=1).
func DisableColor() {
	color.NoColor = true
}

// EnableColor allows colors unless NO_COLOR or CI is set.
func EnableColor() {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("CI") != "" {
		return
	}
	color.NoColor = false
}

// Section draws a readable phase divider.
func Section(title string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}
	title = strings.ToUpper(title)
	const maxTitle = 40
	if len(title) > maxTitle {
		title = title[:maxTitle]
	}
	pad := 64 - len(title) - 4
	if pad < 8 {
		pad = 8
	}
	line := strings.Repeat("━", pad)
	fmt.Fprintln(color.Output)
	if color.NoColor {
		fmt.Fprintf(color.Output, "── %s %s\n", title, strings.Repeat("-", pad))
		return
	}
	secTitle.Fprintf(color.Output, "── %s ", title)
	secLine.Fprintf(color.Output, "%s\n", line)
}

// Step emits a compact sub-phase line (within a section).
func Step(label string) {
	label = strings.TrimSpace(label)
	if label == "" {
		return
	}
	fmt.Fprintln(color.Output)
	if color.NoColor {
		fmt.Fprintf(color.Output, "%s %s\n", stamp(), label)
		return
	}
	fmt.Fprint(color.Output, stamp())
	stepArrow.Fprint(color.Output, "▸ ")
	stepLabel.Fprintf(color.Output, "%s\n", label)
}

func printPrefixed(pref *color.Color, tag, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if color.NoColor {
		fmt.Fprintf(color.Output, "%s%s %s\n", stamp(), tag, msg)
		return
	}
	fmt.Fprint(color.Output, stamp())
	pref.Fprintf(color.Output, "%s ", tag)
	fmt.Fprintln(color.Output, msg)
}

// Infof reports normal progress.
func Infof(format string, args ...any) {
	printPrefixed(labelInfo, "INFO", format, args...)
}

// Notef is an informational line without the INFO tag (e.g. resolved paths).
func Notef(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprint(color.Output, stamp())
	if color.NoColor {
		fmt.Fprintln(color.Output, msg)
		return
	}
	emHighlight.Fprintln(color.Output, msg)
}

// Warnf reports a non-fatal issue.
func Warnf(format string, args ...any) {
	printPrefixed(labelWarn, "WARN", format, args...)
}

// Errorf reports a recoverable error line.
func Errorf(format string, args ...any) {
	printPrefixed(labelErr, "ERR ", format, args...)
}

// Okf reports success for an operation.
func Okf(format string, args ...any) {
	printPrefixed(labelOK, "OK  ", format, args...)
}

// SetDebug toggles DBG lines from Debugf (--debug CLI).
func SetDebug(on bool) {
	debugOn = on
}

// IsDebug reports whether DBG output is enabled.
func IsDebug() bool {
	return debugOn
}

// Debugf emits a magenta DEBUG line when SetDebug(true).
func Debugf(format string, args ...any) {
	if !debugOn {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprint(color.Output, stamp())
	if color.NoColor {
		fmt.Fprintf(color.Output, "DBG   %s\n", msg)
		return
	}
	labelDebug.Fprint(color.Output, "DBG   ")
	fmt.Fprintln(color.Output, msg)
}

// PlanItem prints one row in the publish plan (index, package, version).
func PlanItem(i int, pkg, ver string) {
	fmt.Fprint(color.Output, stamp())
	if color.NoColor {
		fmt.Fprintf(color.Output, "     %3d) %s @ %s\n", i, pkg, ver)
		return
	}
	labelInfo.Fprint(color.Output, "     ")
	idxMuted.Fprintf(color.Output, "%3d) ", i)
	emPkg.Fprint(color.Output, pkg)
	linkMuted.Fprint(color.Output, " @ ")
	emVer.Fprintln(color.Output, ver)
}

// PlanWarn prints a planner warning line.
func PlanWarn(msg string) {
	fmt.Fprint(color.Output, stamp())
	if color.NoColor {
		fmt.Fprintf(color.Output, "PLAN WARN: %s\n", msg)
		return
	}
	labelWarn.Fprint(color.Output, "PLAN WARN ")
	fmt.Fprintln(color.Output, msg)
}

// Done prints the final success banner.
func Done() {
	fmt.Fprintln(color.Output)
	if color.NoColor {
		fmt.Fprintf(color.Output, "%sDone.\n", stamp())
		return
	}
	fmt.Fprint(color.Output, stamp())
	labelOK.Fprint(color.Output, "✓ ")
	emHighlight.Fprintln(color.Output, "Done.")
}

// Fatal prints err and exits with code 1.
func Fatal(err error) {
	if err == nil {
		return
	}
	fmt.Fprint(color.Output, stamp())
	if color.NoColor {
		fmt.Fprintf(color.Output, "FATAL: %v\n", err)
	} else {
		labelErr.Fprint(color.Output, "FATAL ")
		fmt.Fprintf(color.Output, "%v\n", err)
	}
	os.Exit(1)
}

// Fatalf prints formatted message and exits with code 1.
func Fatalf(format string, args ...any) {
	fmt.Fprint(color.Output, stamp())
	if color.NoColor {
		fmt.Fprintf(color.Output, "FATAL: "+format+"\n", args...)
	} else {
		labelErr.Fprint(color.Output, "FATAL ")
		fmt.Fprintf(color.Output, format+"\n", args...)
	}
	os.Exit(1)
}

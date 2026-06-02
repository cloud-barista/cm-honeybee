package software

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const versionDetectTimeout = 5 * time.Second

var versionFlags = []string{"--version", "-version", "-v", "version"}

// versionRegex matches a dotted version token (at least major.minor), e.g.
// "10.1.34", "24.0.1". Bare single integers are intentionally not matched to
// avoid false positives.
var versionRegex = regexp.MustCompile(`[0-9]+\.[0-9]+(\.[0-9]+)*`)

// detectBinaryVersion determines a software version using, in priority order:
//
//	(C) executing the binary with a version flag, force-killed on timeout;
//	(B) reading known application metadata files (JDK release, Tomcat RELEASE-NOTES);
//	(A) parsing a version token from the install path.
//
// Returns "" when no version could be determined.
func detectBinaryVersion(exePath string, cmdSlice []string, environ []string) string {
	if v := versionByExec(exePath); v != "" {
		return v
	}
	if v := versionByMetadata(cmdSlice, environ); v != "" {
		return v
	}
	return versionByPath(exePath, cmdSlice, environ)
}

// versionByExec runs the executable with common version flags. Each invocation is
// bounded by a timeout; if the process does not finish in time it is force-killed
// (CommandContext sends SIGKILL on timeout, and WaitDelay closes inherited pipes so
// a lingering child cannot block us).
func versionByExec(exePath string) string {
	if exePath == "" {
		return ""
	}

	for _, flag := range versionFlags {
		ctx, cancel := context.WithTimeout(context.Background(), versionDetectTimeout)
		cmd := exec.CommandContext(ctx, exePath, flag)
		cmd.WaitDelay = 2 * time.Second
		out, _ := cmd.CombinedOutput()
		cancel()

		if v := versionRegex.FindString(string(out)); v != "" {
			return v
		}
	}

	return ""
}

// versionByMetadata reads version information from well-known application files.
func versionByMetadata(cmdSlice []string, environ []string) string {
	// JDK: $JAVA_HOME/release -> JAVA_VERSION="24.0.1"
	if javaHome := javaHomeFrom(environ, ""); javaHome != "" {
		if data, err := os.ReadFile(filepath.Join(javaHome, "release")); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "JAVA_VERSION=") {
					if v := versionRegex.FindString(line); v != "" {
						return v
					}
				}
			}
		}
	}

	// Tomcat: <catalina.home>/RELEASE-NOTES -> "Apache Tomcat Version 10.1.34"
	if home := catalinaHomeFrom(cmdSlice); home != "" {
		if data, err := os.ReadFile(filepath.Join(home, "RELEASE-NOTES")); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.Contains(line, "Apache Tomcat Version") {
					if v := versionRegex.FindString(line); v != "" {
						return v
					}
				}
			}
		}
	}

	return ""
}

// versionByPath parses a version token from the install directory name, resolving
// symlinks first (e.g. /opt/tomcat/latest -> apache-tomcat-10.1.34 -> "10.1.34").
func versionByPath(exePath string, cmdSlice []string, environ []string) string {
	candidates := []string{
		catalinaHomeFrom(cmdSlice),
		javaHomeFrom(environ, exePath),
		filepath.Dir(exePath),
	}

	for _, c := range candidates {
		if c == "" {
			continue
		}

		real := c
		if r, err := filepath.EvalSymlinks(c); err == nil {
			real = r
		}

		segments := strings.Split(real, string(os.PathSeparator))
		for i := len(segments) - 1; i >= 0; i-- {
			if v := versionRegex.FindString(segments[i]); v != "" {
				return v
			}
		}
	}

	return ""
}

// catalinaHomeFrom derives Tomcat's install dir from -Dcatalina.home (or
// -Dcatalina.base) in a JVM command line. Returns "" if absent.
func catalinaHomeFrom(cmdSlice []string) string {
	var home, base string
	for _, arg := range cmdSlice {
		if v, ok := strings.CutPrefix(arg, "-Dcatalina.home="); ok {
			home = strings.TrimSpace(v)
		} else if v, ok := strings.CutPrefix(arg, "-Dcatalina.base="); ok {
			base = strings.TrimSpace(v)
		}
	}
	if home != "" {
		return home
	}
	return base
}

// javaHomeFrom derives the JDK/JRE home from JAVA_HOME, falling back to the java
// executable path (<javaHome>/bin/java). Returns "" if not a JVM.
func javaHomeFrom(environ []string, exePath string) string {
	for _, e := range environ {
		if v, ok := strings.CutPrefix(e, "JAVA_HOME="); ok {
			if v = strings.TrimSpace(v); v != "" {
				return v
			}
		}
	}

	if exePath != "" {
		switch filepath.Base(exePath) {
		case "java", "java.exe":
			binDir := filepath.Dir(exePath)
			if filepath.Base(binDir) == "bin" {
				return filepath.Dir(binDir)
			}
		}
	}

	return ""
}

package software

import (
	"archive/zip"
	"context"
	"io"
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

// detectBinaryVersion determines a software version. It first branches on the
// software identity, because the right source of a version depends on what the
// software is:
//
//   - JVM applications run via the `java` executable, so `java -version` would
//     report the JDK version rather than the application's. Tomcat is identified
//     via -Dcatalina.home and its version is read from catalina.jar / RELEASE-NOTES;
//     other Java apps fall back to the JDK version.
//   - Native binaries use exec (`--version`) then an install-path token.
//
// Returns "" when no version could be determined.
func detectBinaryVersion(exePath string, cmdSlice []string, environ []string) string {
	if isJavaProcess(exePath, cmdSlice) {
		return detectJavaAppVersion(exePath, cmdSlice, environ)
	}

	if v := versionByExec(exePath); v != "" {
		return v
	}
	return versionFromPath(filepath.Dir(exePath))
}

// isJavaProcess reports whether the process runs on the JVM (executable or argv[0]
// is java).
func isJavaProcess(exePath string, cmdSlice []string) bool {
	if isJavaBinary(exePath) {
		return true
	}
	if len(cmdSlice) > 0 {
		return isJavaBinary(cmdSlice[0])
	}
	return false
}

func isJavaBinary(p string) bool {
	switch filepath.Base(p) {
	case "java", "java.exe":
		return true
	}
	return false
}

// detectJavaAppVersion resolves the version of a JVM-hosted application: Tomcat if
// catalina.home is present, otherwise the backing JDK version.
func detectJavaAppVersion(exePath string, cmdSlice []string, environ []string) string {
	if home := catalinaHomeFrom(cmdSlice); home != "" {
		if v := tomcatVersion(home); v != "" {
			return v
		}
		if v := versionFromPath(home); v != "" {
			return v
		}
	}

	if javaHome := javaHomeFrom(environ, exePath); javaHome != "" {
		if v := jdkVersion(javaHome); v != "" {
			return v
		}
		if v := versionFromPath(javaHome); v != "" {
			return v
		}
	}

	// Last resort: ask the JVM itself.
	return versionByExec(exePath)
}

// tomcatVersion reads the Tomcat version from its install dir, preferring the
// authoritative server.number in catalina.jar, then RELEASE-NOTES.
func tomcatVersion(catalinaHome string) string {
	if v := tomcatVersionFromJar(filepath.Join(catalinaHome, "lib", "catalina.jar")); v != "" {
		return v
	}

	if data, err := os.ReadFile(filepath.Join(catalinaHome, "RELEASE-NOTES")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "Apache Tomcat Version") {
				if v := versionRegex.FindString(line); v != "" {
					return v
				}
			}
		}
	}

	return ""
}

// tomcatVersionFromJar reads server.number from
// org/apache/catalina/util/ServerInfo.properties inside catalina.jar.
func tomcatVersionFromJar(jarPath string) string {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return ""
	}
	defer func() {
		_ = r.Close()
	}()

	for _, f := range r.File {
		if f.Name != "org/apache/catalina/util/ServerInfo.properties" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return ""
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "server.number=") {
				return versionRegex.FindString(line)
			}
		}
	}

	return ""
}

// jdkVersion reads JAVA_VERSION from <javaHome>/release.
func jdkVersion(javaHome string) string {
	data, err := os.ReadFile(filepath.Join(javaHome, "release"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "JAVA_VERSION=") {
			if v := versionRegex.FindString(line); v != "" {
				return v
			}
		}
	}
	return ""
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

// versionFromPath parses a version token from a directory name, resolving symlinks
// first (e.g. /opt/tomcat/latest -> apache-tomcat-10.1.34 -> "10.1.34").
func versionFromPath(dir string) string {
	if dir == "" {
		return ""
	}

	real := dir
	if r, err := filepath.EvalSymlinks(dir); err == nil {
		real = r
	}

	segments := strings.Split(real, string(os.PathSeparator))
	for i := len(segments) - 1; i >= 0; i-- {
		if v := versionRegex.FindString(segments[i]); v != "" {
			return v
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

	if exePath != "" && isJavaBinary(exePath) {
		binDir := filepath.Dir(exePath)
		if filepath.Base(binDir) == "bin" {
			return filepath.Dir(binDir)
		}
	}

	return ""
}

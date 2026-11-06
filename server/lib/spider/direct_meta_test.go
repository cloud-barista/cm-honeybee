package spider

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"testing"

	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
)

// metaChildEnv makes TestCloudOSMetaInfoFirstLoad run as a child process.
// cb-spider caches the metadata in a package variable, so only a fresh process
// can observe the first load; the parent test re-runs this binary for that.
const metaChildEnv = "HB_SPIDER_META_CHILD"

const metaLoaders = 64

var emptyLoadsLine = regexp.MustCompile(`(?m)^EMPTY_LOADS=(\d+)$`)

// TestCloudOSMetaInfoFirstLoad shows that concurrent first loads of
// cloudos_meta.yaml can return empty metadata, and that
// PreloadCloudOSMetaInfo prevents it.
func TestCloudOSMetaInfoFirstLoad(t *testing.T) {
	if mode := os.Getenv(metaChildEnv); mode != "" {
		loadMetaConcurrently(t, mode == "preload")
		return
	}

	// Without the preload: the race depends on scheduling, so several fresh
	// processes are tried and one that sees an empty load is enough.
	const tries = 20
	raced := 0
	for i := 0; i < tries && raced == 0; i++ {
		raced = runMetaChild(t, "race")
	}
	if raced == 0 {
		t.Logf("no empty load in %d processes without the preload; the race was not reproduced on this run", tries)
	} else {
		t.Logf("without the preload, %d of %d concurrent first loads got empty metadata", raced, metaLoaders)
	}

	for i := 0; i < tries; i++ {
		if n := runMetaChild(t, "preload"); n != 0 {
			t.Fatalf("with the preload, %d of %d concurrent loads got empty metadata", n, metaLoaders)
		}
	}
}

func runMetaChild(t *testing.T, mode string) int {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestCloudOSMetaInfoFirstLoad$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), metaChildEnv+"="+mode)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child (%s): %v\n%s", mode, err, out)
	}
	m := emptyLoadsLine.FindSubmatch(out)
	if m == nil {
		t.Fatalf("child (%s) printed no result:\n%s", mode, out)
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

func loadMetaConcurrently(t *testing.T, preload bool) {
	if preload {
		if err := PreloadCloudOSMetaInfo(); err != nil {
			t.Fatal(err)
		}
	}

	var (
		start sync.WaitGroup
		done  sync.WaitGroup
		mu    sync.Mutex
		empty int
	)
	start.Add(1)
	for i := 0; i < metaLoaders; i++ {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			meta, err := cim.GetCloudOSMetaInfo("AWS")
			if err != nil || len(meta.CredentialCSP) == 0 {
				mu.Lock()
				empty++
				mu.Unlock()
			}
		}()
	}
	start.Done()
	done.Wait()
	fmt.Printf("EMPTY_LOADS=%d\n", empty)
}

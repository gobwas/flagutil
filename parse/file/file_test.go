package file

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/gobwas/flagutil/parse/testutil"
)

var (
	_ Lookup = MultiLookup{}
	_ Lookup = FlagLookup("")
	_ Lookup = PathLookup{}
	_ Lookup = BytesLookup{}
)

func TestPathLookup(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)

	t.Run("fail dir", func(t *testing.T) {
		lookup := PathLookup{dir}
		if _, err := lookup.Lookup(nil); err == nil {
			t.Fatal("want error; got nil")
		}
	})
	t.Run("success", func(t *testing.T) {
		file, err := os.Create(path.Join(dir, "config"))
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file.Name())
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}

		lookup := PathLookup{file.Name()}
		rc, err := lookup.Lookup(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rc.Close()
	})
}

func TestFlagLookup(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	exp := make([]byte, 512)
	n, err := rand.Read(exp)
	if err != nil {
		t.Fatal(err)
	}
	exp = exp[:n]

	if _, err := file.Write(exp); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	lookup := FlagLookup("config")

	fs := new(testutil.StubFlagSet)
	fs.AddFlag("config", file.Name())

	rc, err := lookup.Lookup(fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	act, err := ioutil.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(act, exp) {
		t.Fatalf("unexpected file contents")
	}
}

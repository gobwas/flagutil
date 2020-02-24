package file

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"

	"github.com/gobwas/flagutil/parse/testutil"
)

var (
	_ Lookup = MultiLookup{}
	_ Lookup = FlagLookup("")
	_ Lookup = PathLookup("")
	_ Lookup = BytesLookup{}
)

func TestPathLookupDir(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	lookup := PathLookup(dir)
	if _, err := lookup.Lookup(nil); err == nil {
		t.Fatal("want error; got nil")
	}
}

func TestPathLookup(t *testing.T) {
	file, exp, err := tempFile()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	lookup := PathLookup(file.Name())
	rc, err := lookup.Lookup(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	act, err := ioutil.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(act, exp) {
		t.Fatalf("unexpected file contents")
	}
}

func TestFlagLookup(t *testing.T) {
	file, exp, err := tempFile()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	lookup := FlagLookup("config")

	fs := new(testutil.StubFlagSet)
	fs.AddFlag("config", file.Name())

	rc, err := lookup.Lookup(fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	act, err := ioutil.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(act, exp) {
		t.Fatalf("unexpected file contents")
	}
}

func tempFile() (file *os.File, content []byte, err error) {
	file, err = ioutil.TempFile("", "")
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			file.Close()
			os.Remove(file.Name())
		}
	}()

	content = make([]byte, 512)
	n, err := rand.Read(content)
	if err != nil {
		return nil, nil, err
	}
	content = content[:n]

	if _, err := file.Write(content); err != nil {
		return nil, nil, err
	}
	if err := file.Close(); err != nil {
		return nil, nil, err
	}

	return file, content, nil
}

//go:build mage
// +build mage

package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/magefile/mage/sh"
)

type mutation struct {
	// Copy files
	From, To string

	// Go Replace by regex
	Match, Replace, Glob string

	DeleteGlob string
}

type DependencyGithub struct {
	Repo   string
	Branch string
}

var (
	deps = []DependencyGithub{
		{Repo: "https://github.com/Velocidex/go-prefetch"},
		{Repo: "https://github.com/Velocidex/etw"},
		{
			Repo:   "https://github.com/Velocidex/velociraptor",
			Branch: "v0.75-release",
		},
	}

	// The Version of Go to use that still works on legacy systems.
	golang_url = map[string]string{
		"linux":  "https://go.dev/dl/go1.24.10.linux-amd64.tar.gz",
		"darwin": "https://go.dev/dl/go1.24.10.darwin-arm64.tar.gz",
	}

	node_url = map[string]string{
		"linux":  "https://nodejs.org/dist/v24.11.1/node-v24.11.1-linux-x64.tar.gz",
		"darwin": "https://nodejs.org/dist/v24.11.1/node-v24.11.1-darwin-arm64.tar.gz",
	}

	build_targets = map[string][]string{
		"linux": []string{
			"UpdateDependentTools", "Assets", "Linux"},

		"darwin": []string{
			"UpdateDependentTools", "Assets", "Darwin", "DarwinM1"},
	}

	// Transform the codebase so it can build
	mutations = []mutation{
		{From: "../patches/velociraptor/go.mod", To: "velociraptor/go.mod"},
		{From: "../patches/velociraptor/go.sum", To: "velociraptor/go.sum"},
		{From: "../patches/velociraptor/host_darwin_cgo.go",
			To: "velociraptor/vql/psutils/host_darwin_cgo.go"},
		{From: "../patches/etw/go.mod", To: "etw/go.mod"},
		{From: "../patches/prefetch/go.mod",
			To: "go-prefetch/go.mod"},
		{From: "../patches/json/validator.go",
			To: "velociraptor/tools/json/validator.go"},
	}
)

func installGo() error {
	return installPackageFromURL(golang_url[runtime.GOOS], "go")
}

func installNode() error {
	return installPackageFromURL(node_url[runtime.GOOS], "node")
}

func installPackageFromURL(url string, dst string) error {
	stat, err := os.Lstat(dst)
	if err == nil && stat.Mode().IsDir() {
		return nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case header == nil:
			continue
		}

		// Remove the top level directory from the zip.
		components := strings.Split(header.Name, "/")
		if len(components) <= 1 {
			continue
		}
		target := filepath.Join(dst, strings.Join(components[1:], "/"))

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			err := os.MkdirAll(target, 0755)
			if err != nil {
				return err
			}

		case tar.TypeReg:
			fmt.Printf("Creating %v (%v bytes)\n", target, header.Size)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			_, err = io.Copy(f, tr)
			if err != nil {
				return err
			}

			f.Close()

			err = os.Chmod(target, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
		}
	}
}

func replace_string_in_file(filename string, old string, new string) error {
	read, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	newContents := strings.Replace(string(read), old, new, -1)
	return ioutil.WriteFile(filename, []byte(newContents), 0644)
}

func maybeClone(dep DependencyGithub) error {
	base := filepath.Base(dep.Repo)
	_, err := os.Lstat(base)
	if err == nil {
		return nil
	}

	branch := "master"
	if dep.Branch != "" {
		branch = dep.Branch
	}

	return sh.RunV("git", "clone", "--depth", "1",
		"--single-branch", "-b", branch, dep.Repo)
}

func copyOutput(toplevel string) error {
	err := os.MkdirAll(toplevel+"/output/", 0700)
	if err != nil {
		return err
	}

	basepath := toplevel + "/build/velociraptor/output/"
	matches, err := os.ReadDir(basepath)
	if err != nil {
		return err
	}

	for _, match := range matches {
		filename := filepath.Join(basepath, match.Name())

		fmt.Printf("Found output file %v\n", filename)

		dst := toplevel + "/output/" + match.Name() + "-legacy"
		fmt.Printf("Copying %v in %v\n", filename, dst)
		err := sh.Copy(dst, filename)
		if err != nil {
			return err
		}
	}
	return nil
}

func Build() error {
	err := os.MkdirAll("build", 0700)
	if err != nil {
		return err
	}

	toplevel, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(toplevel)

	err = os.Chdir("build")
	if err != nil {
		return err
	}

	err = installGo()
	if err != nil {
		return err
	}

	err = installNode()
	if err != nil {
		return err
	}

	for _, dep := range deps {
		err = maybeClone(dep)
		if err != nil {
			return err
		}
	}

	for _, m := range mutations {
		if m.From != "" {
			fmt.Printf("Copying %v to %v\n", m.From, m.To)
			basedir := filepath.Dir(m.To)
			os.MkdirAll(basedir, 0755)

			err := sh.Copy(m.To, m.From)
			if err != nil {
				return err
			}
		}

		if m.DeleteGlob != "" {
			basepath, pattern := doublestar.SplitPattern(m.DeleteGlob)
			fsys := os.DirFS(basepath)
			matches, err := doublestar.Glob(fsys, pattern)
			if err != nil {
				return err
			}

			for _, match := range matches {
				filename := filepath.Join(basepath, match)
				fmt.Printf("Deleting %v in %v\n", m.Match, filename)
				err = os.Remove(filename)
				if err != nil {
					return err
				}
			}
		}

		if m.Glob != "" {
			basepath, pattern := doublestar.SplitPattern(m.Glob)
			fsys := os.DirFS(basepath)
			matches, err := doublestar.Glob(fsys, pattern)
			if err != nil {
				return err
			}

			for _, match := range matches {
				filename := filepath.Join(basepath, match)
				fmt.Printf("Replacing %v in %v\n", m.Match, filename)
				err = replace_string_in_file(filename, m.Match, m.Replace)
				if err != nil {
					return err
				}
			}
		}

	}

	// Build steps
	err = os.Chdir("velociraptor")
	if err != nil {
		return err
	}

	env := make(map[string]string)
	env["PATH"] = toplevel + "/build/go/bin/:" +
		toplevel + "/build/node/bin/:" + os.Getenv("PATH")
	env["GOPATH"] = ""

	fmt.Printf("Checking PATH: %v\n", env["PATH"])

	// Prevent automatic toolchain switching.
	env["GOTOOLCHAIN"] = "local"

	go_path := toplevel + "/build/go/bin/go"
	env["MAGEFILE_GOCMD"] = go_path
	env["MAGEFILE_PATH"] = env["PATH"]
	env["MAGEFILE_VERBOSE"] = "1"

	fmt.Printf("Checking go version:\n")
	err = sh.RunWithV(env, "which", "go")
	if err != nil {
		return err
	}

	err = sh.RunWithV(env, "go", "version")
	if err != nil {
		return err
	}

	fmt.Printf("Checking node version:\n")
	err = sh.RunWithV(env, "node", "--version")
	if err != nil {
		return err
	}

	for _, target := range build_targets[runtime.GOOS] {
		err = sh.RunWithV(env, go_path, "run", "-v", "./make.go", target)
		if err != nil {
			return err
		}
	}

	err = copyOutput(toplevel)
	if err != nil {
		return err
	}

	return nil
}

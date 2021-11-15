/*
Create Luet repositories on github

*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	luetClient "github.com/mudler/luet/pkg/api/client"
	utils "github.com/mudler/luet/pkg/api/client/utils"
)

type opData struct {
	FinalRepo string
}

type resultData struct {
	Package luetClient.Package
	Exists  bool
}

var buildPackages = flag.Bool("build", false, "Build missing packages, or specified")
var download = flag.Bool("downloadMeta", false, "Download packages metadata")

var push = os.Getenv("PUSH_CACHE")

var createRepo = flag.Bool("create-repo", false, "create repository")
var tree = flag.String("tree", "${PWD}/packages", "create repository")

//goaction:description Final container registry repository
var finalRepo = os.Getenv("FINAL_REPO")

//goaction:description Current package to build
var currentPackage = os.Getenv("CURRENT_PACKAGE")

//goaction:description Repository Name
var repositoryName = os.Getenv("REPOSITORY_NAME")

func main() {
	flag.Parse()
	utils.RunSH("dependencies", "apk add curl")
	utils.RunSH("dependencies", "curl https://get.mocaccino.org/luet/get_luet_root.sh | sh")
	utils.RunSH("dependencies", "luet install -y utils/jq")

	switch {
	case *buildPackages:
		build()
	case *createRepo:
		create()
	case *download:
		downloadMeta()
	}
}

func metaWorker(i int, wg *sync.WaitGroup, c <-chan luetClient.Package, o opData) error {
	defer wg.Done()

	for p := range c {
		tmpdir, err := ioutil.TempDir(os.TempDir(), "ci")
		checkErr(err)
		unpackdir, err := ioutil.TempDir(os.TempDir(), "ci")
		checkErr(err)
		utils.RunSH("unpack", fmt.Sprintf("TMPDIR=%s XDG_RUNTIME_DIR=%s luet util unpack %s %s", tmpdir, tmpdir, p.ImageMetadata(o.FinalRepo), unpackdir))
		utils.RunSH("move", fmt.Sprintf("mv %s/* build/", unpackdir))
		checkErr(err)
		os.RemoveAll(tmpdir)
		os.RemoveAll(unpackdir)
	}
	return nil
}

func buildWorker(i int, wg *sync.WaitGroup, c <-chan luetClient.Package, o opData, results chan<- resultData) error {
	defer wg.Done()

	for p := range c {
		fmt.Println("Checking", p)
		results <- resultData{Package: p, Exists: p.ImageAvailable(o.FinalRepo)}
	}
	return nil
}

func create() {
	if push == "true" {
		utils.RunSH(
			"create_repo",
			fmt.Sprintf(
				"sudo -E luet create-repo --name '%s' --packages %s --tree %s --push-images --type docker 	--output %s",
				repositoryName, "${PWD}/build", *tree, finalRepo,
			),
		)
	} else {
		utils.RunSH(
			"create_repo",
			fmt.Sprintf(
				"sudo -E luet create-repo --name '%s' --packages %s --tree %s --type http --output ${PWD}/build",
				repositoryName, "${PWD}/build", *tree,
			),
		)
	}
}

func build() {
	packs, err := luetClient.TreePackages(*tree)
	checkErr(err)

	for _, p := range packs.Packages {
		if currentPackage != "" && p.EqualSV(currentPackage) && !p.ImageAvailable(finalRepo) ||
			currentPackage == "" {
			fmt.Println("Building", p.String())
			checkErr(utils.RunSH("build", fmt.Sprintf("./build.sh %s", p.String())))
		}
	}
}

func downloadMeta() {
	packs, err := luetClient.TreePackages(*tree)
	checkErr(err)

	all := make(chan luetClient.Package)
	wg := new(sync.WaitGroup)

	for i := 0; i < 1; i++ {
		wg.Add(1)
		go metaWorker(i, wg, all, opData{FinalRepo: finalRepo})
	}

	for _, p := range packs.Packages {
		all <- p
	}
	close(all)
	wg.Wait()
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

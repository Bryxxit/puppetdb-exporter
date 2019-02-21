package cornelius

import (
	"fmt"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"strings"

	"gopkg.in/src-d/go-git.v4"
)

// func CheckIfGit(path string)  bool{
// 	if _, err := os.Stat(path); os.IsNotExist(err) {
// 		// path/to/whatever does not exist
// 		return false
// 	}

// 	return true

// }

type PModule struct {
	Name    string
	Version string
	Git     string
	Commit  string
}

type PuppetFile struct {
	Modules []PModule `yaml:"modules"`
}

func LoadEnvFromFile(path string, cacheDir string, destDir string) {
	modules, err := loadModulesFromFile(path)
	if err != nil {
		//log.Fatal(err)
		println(err)
	}
	loadModules(modules, cacheDir)
	deployModules(modules, cacheDir, destDir)

}

func loadModules(modules []PModule, cacheDir string) {
	for _, element := range modules {
		//println(element.Name)
		if strings.Contains(element.Name, "-") {
			m, err := GetModuleInfo(element.Name)
			if err == nil {
				err := CacheRepo(m.CurrentRelease.Metadata.Source, cacheDir)
				println(err)
			} else {
				println(err)
			}

		} else if element.Git != "" {
			err := CacheRepo(element.Git, cacheDir)
			println(err)
		}
	}
}

func deployModules(modules []PModule, cacheDir string, destdir string) {
	for _, element := range modules {
		s := element.Name
		n := element.Name
		if strings.Contains(element.Name, "-") {
			urlArr := strings.Split(element.Name, "-")
			n = urlArr[len(urlArr)-1]
		} else if element.Git != "" {
			urlArr := strings.Split(element.Git, "/")
			s = strings.Replace(urlArr[len(urlArr)-1], ".git", "", -1)
		}

		println(s, n)
		err := CloneRepo(cacheDir+"/"+s, destdir+"/"+n)
		if element.Version != "" {
			//println(element.Version)
			checkoutTag(destdir+"/"+n, "v"+element.Version)
		}
		println(err)
	}
}

func checkOutBranch(repo string, tag string) {
	r, err := git.PlainOpen(repo)
	println(err)
	tagrefs, err := r.Branches()
	var h *plumbing.Reference
	err = tagrefs.ForEach(func(t *plumbing.Reference) error {
		if t.Name().Short() == tag {
			h = t
		}
		return nil
	})

	//println(err, h.Hash().String())
	if h != nil {
		ref, err := r.Head()
		w, err := r.Worktree()
		err = w.Checkout(&git.CheckoutOptions{
			Hash: h.Hash(),
		})

		ref, err = r.Head()
		fmt.Println(ref.Hash())
		println(err)
	}
}

func checkoutTag(repo string, tag string) {
	r, err := git.PlainOpen(repo)
	println(err)
	tagrefs, err := r.Tags()
	var h *plumbing.Reference
	err = tagrefs.ForEach(func(t *plumbing.Reference) error {
		if t.Name().Short() == tag {
			h = t
		}
		return nil
	})

	//println(err, h.Hash().String())
	if h != nil {
		ref, err := r.Head()
		w, err := r.Worktree()
		err = w.Checkout(&git.CheckoutOptions{
			Hash: h.Hash(),
		})

		ref, err = r.Head()
		fmt.Println(ref.Hash())
		println(err)
	}

}

func loadModulesFromFile(path string) ([]PModule, error) {
	fileLoc := path

	f, err := os.Open(fileLoc)
	if err != nil {
		log.Fatalf("os.Open() failed with '%s'\n", err)
		return nil, err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)

	var yamlFile PuppetFile
	err = dec.Decode(&yamlFile)
	if err != nil {
		log.Fatalf("dec.Decode() failed with '%s'\n", err)
		return nil, err
	}
	return yamlFile.Modules, err

}

// CacheRepo Caches the repo
func CacheRepo(url string, cacheDir string) error {
	urlArr := strings.Split(url, "/")
	dirName := strings.Replace(cacheDir+"/"+urlArr[len(urlArr)-1], ".git", "", -1)
	err := CloneRepo(url, dirName)
	return err

}

func CloneRepo(url string, dest string) error {
	_, err := git.PlainClone(dest, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})

	if err != nil {
		errString := err.Error()
		if errString == "repository already exists" {
			r, err := git.PlainOpen(dest)
			w, err := r.Worktree()
			err = w.Pull(&git.PullOptions{})
			if err != nil {
				errString = err.Error()
				if errString != "already up-to-date" {
					return err
				}
				return nil
			}
		}

	}
	return err
}

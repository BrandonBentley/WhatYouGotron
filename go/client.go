package main

import (
	"fmt"
	"os"
	"path/filepath"
	//"errors"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
	//"bufio"
	"regexp"
	"sync"
	//	"net/http"
	//	"bytes"
	"io/ioutil"
)

var root Folder
var rootIndex int
var current *Folder
var rootPath string
var fileCount int
var folderCount int
var failedItems []FailItem

type FailItem struct {
	Path    string
	itemErr error
}
type File struct {
	Name     string
	FullPath string
}

type Folder struct {
	Name     string
	FullPath string
	Files    []File
	Folders  []Folder
}

type OverHeadPackage struct {
	FileCount   int
	FolderCount int
	Root        Folder
}

func (f *Folder) AddFile(name string, path string) {
	f.Files = append(f.Files, CreateFile(name, path))
}

func (f *Folder) AddFolder(name string, path string) {
	f.Folders = append(f.Folders, CreateFolder(name, path))
}

func (f *Folder) FindSubFolderRecursive(path string) (*Folder, error) {
	folder := f

	//fmt.Println("==========Search for Directory==========")
	//fmt.Println("Searching for " + path + " in " + folder.FullPath)
	paths := strings.Split(path, "\\")
	paths = append(paths[:0], paths[1:]...)
	for _, s := range paths {
		//fmt.Println("searching " + s)
		tempFolder := folder.FindSubFolder(s)
		if tempFolder != nil {
			folder = tempFolder
		}
	}
	//fmt.Println("\t\t" + strconv.FormatBool(folder == nil))
	return folder, nil
}

func (f *Folder) FindSubFolder(folderName string) *Folder {
	found := false
	//fmt.Println("looking for: " + folderName)
	//fmt.Print("Contains: ")
	for i := 0; i < len(f.Folders); i++ {
		//fmt.Print(f.Folders[i].Name + " ")
		if f.Folders[i].Name == folderName {
			//fmt.Println("\nfound: " + folderName)
			found = true
			return &f.Folders[i]
		}
	}
	//fmt.Print("\n")
	if !found {
		//fmt.Println("Folder Not Found")
	}
	return nil
}

func CreateFolder(name string, path string) Folder {
	folderCount++
	return Folder{Name: name, FullPath: path + "\\", Files: make([]File, 0), Folders: make([]Folder, 0)}
}

func CreateFile(name string, path string) File {
	fileCount++
	return File{Name: name, FullPath: path}
}

func visit(path string, f os.FileInfo, err error) error {
	dir, name := filepath.Split(path)
	//fmt.Println(path)
	path = strings.Replace(path, rootPath, root.Folders[rootIndex].FullPath, 1)
	dir = strings.Replace(dir, rootPath, root.Folders[rootIndex].FullPath, 1)
	if f == nil {
		failedItems = append(failedItems, FailItem{Path: path, itemErr: err})
		return err
	}
	if f.IsDir() {
		if dir == "" {
			_, name = filepath.Split(rootPath[:len(rootPath)-1])
			root.Folders[rootIndex] = CreateFolder(name, path)
			current = &root.Folders[rootIndex]
		} else if current.FullPath == dir {
			current.AddFolder(name, path)
			current = &current.Folders[len(current.Folders)-1]
		} else {
			current, err = root.Folders[rootIndex].FindSubFolderRecursive(dir)
			if err != nil {
				log.Fatal(err)
			} else {
				current.AddFolder(name, path)
				current = &current.Folders[len(current.Folders)-1]
			}
		}
	} else {
		if current.FullPath == dir {
			current.AddFile(name, path)
		} else {
			current, err = root.Folders[rootIndex].FindSubFolderRecursive(dir)
			if err != nil {
				log.Fatal(err)
			} else {
				current.AddFile(name, path)
			}
		}

	}

	return nil
}

func initializeCounts() {
	fileCount = 0
	folderCount = 0
}

func progressOutput(totalFolders int, totalFiles int, running *bool, outputWG *sync.WaitGroup) {
	for *running {
		fmt.Printf("\rFolders Scanned: %-13d  Files Scanned: %-13d", folderCount-totalFolders, fileCount-totalFiles)
		time.Sleep(1000000000)
	}
	outputWG.Done()
}

func main() {
	args := os.Args
	if len(args) <= 1 {
		args[0] = filepath.Dir(args[0])
		fmt.Println("No Input Provided. Defaulting to current Directory.")
	} else {
		args = append(args[:0], args[1:]...)
	}
	root.Folders = make([]Folder, len(args))
	root.Files = make([]File, 0)
	inFolders := make([]int, len(args))
	inFiles := make([]int, len(args))
	startTime := time.Now()
	initializeCounts()
	rootIndex = 0
	r, _ := regexp.Compile("\\$")
	var wg sync.WaitGroup
	var outputWG sync.WaitGroup
	for i, arg := range args {
		if !r.Match([]byte(arg)) {
			rootPath = arg + "\\"
		} else {
			rootPath = arg
		}

		rootIndex = i
		_, err := os.Stat(rootPath)
		if err != nil {
			fmt.Println("Directory " + rootPath + " Not found. Skipping")
		} else {
			fmt.Println("Scan Path: " + rootPath)
			wg.Add(1)
			outputWG.Add(1)
			running := true
			if i > 0 {
				go progressOutput(inFolders[i-1], inFiles[i-1], &running, &outputWG)
			} else {
				go progressOutput(0, 0, &running, &outputWG)
			}

			go func() {
				filepath.Walk(rootPath, visit)
				wg.Done()
			}()
			wg.Wait()
			running = false
			outputWG.Wait()
			if i > 0 {
				inFolders[i] = folderCount - inFolders[i-1]
				inFiles[i] = fileCount - inFiles[i-1]
			} else {
				inFolders[i] = folderCount
				inFiles[i] = fileCount
			}
			fmt.Printf("\r%100s", "")
			fmt.Printf("\r - Completed\n")
		}
	}

	fmt.Printf("\nRESULTS\n========================================\n"+
		"%-15s|%-10s|%-10s|\n", " Directory", " Folders", " Files")
	for i, fold := range root.Folders {
		fmt.Printf("  %-13s|  %-8d|  %-8d|\n", fold.Name, inFolders[i], inFiles[i])
	}

	var pack OverHeadPackage
	if len(root.Folders) <= 1 {
		pack = OverHeadPackage{FileCount: fileCount, FolderCount: folderCount, Root: root.Folders[0]}
	} else {
		pack = OverHeadPackage{FileCount: fileCount, FolderCount: folderCount, Root: root}
	}

	jsonData, err := json.Marshal(pack)
	if err != nil {
		fmt.Println("JSON Error")
		fmt.Println(err)
		fmt.Println(len(root.Folders))

	} else {
		fileErr := ioutil.WriteFile(".\\app\\assets\\json\\data.json", jsonData, 0644)
		if fileErr != nil {
			fmt.Println("")
			fmt.Println(err)
		}
		/*
			url := "http://localhost:8080/post"
			_, err := http.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Println("")
				fmt.Println(err)
			}
			//req.Header.Set("Content-Type", "application/json")
		*/
		if len(failedItems) > 0 {
			finalFail, err := json.Marshal(failedItems)
			if err != nil {
				fmt.Println("error Encoding failed-items.json")
				fmt.Println(err)
			} else {
				ioutil.WriteFile(".\\failed-items.json", finalFail, 0644)
				fmt.Println("There are Failed items. Check failed-items.json for more information.")
			}

		}

	}
	elapsed := time.Now().Sub(startTime)
	fmt.Print("\nScan Complete\n\nScanned " + strconv.Itoa(folderCount) + " Folders\nScanned " + strconv.Itoa(fileCount) + " Files\nTime Elapsed: ")
	fmt.Print(elapsed.Seconds())
	fmt.Println(" Seconds\n")
	//bufio.NewReader(os.Stdin).ReadBytes('\n')
}

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"archive/zip"

	"io"
	"os"
	"path/filepath"
	"strconv"
)

type JarProperties struct {
	version       string
	versionNumber int
	filePath      string
	packageName   string
	name          string
	vendor        string
	license       string
}

func main() {

	targetDir := "/Users/xcheng/Downloads/userlib"
	files, err := ioutil.ReadDir(targetDir)
	if err != nil {
		log.Fatal(err)
	}

	jars := []JarProperties{}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".jar") {
			//log.Println("Processing " + f.Name())
			filePath := filepath.Join(targetDir, f.Name())
			jarProp := getJarProperties(filePath)
			if strings.Compare(jarProp.filePath, "") != 0 {
				jars = append(jars, jarProp)
				//log.Println(jarProp)
			}
		}
	}

	// remove duplicates
	var keepJars = make(map[string]JarProperties)

	for _, jar1 := range jars {
		//log.Println("Checking " + jar1.filePath)
		if _, ok := keepJars[jar1.packageName]; !ok {
			keepJars[jar1.packageName] = jar1
		}
		packageName := jar1.packageName

		// find latest
		for _, jar2 := range jars {
			latestJar := keepJars[packageName]
			if strings.Compare(jar1.filePath, jar2.filePath) == 0 {
				// skip self
				continue
			}
			if strings.Compare(latestJar.filePath, jar2.filePath) == 0 {
				// skip self
				continue
			}
			if strings.Compare(packageName, jar2.packageName) == 0 {
				goodFileSuffix := fmt.Sprintf("%s%s", jar2.version, ".jar")
				if latestJar.versionNumber == jar2.versionNumber && strings.HasSuffix(jar2.filePath, goodFileSuffix) {
					// same version and has better file name
					log.Println("Better file name: " + jar2.filePath)
					keepJars[packageName] = jar2
				} else if latestJar.versionNumber < jar2.versionNumber {
					// newer version found
					log.Println("Newer version: " + jar2.filePath)
					keepJars[packageName] = jar2
				}
			}
		}
	}

	count := 0
	for _, jar := range jars {
		jarToKeep := keepJars[jar.packageName]
		if strings.Compare(jar.filePath, jarToKeep.filePath) != 0 {
			if _, err := os.Stat(jar.filePath); err == nil {
				log.Println("Removing duplicate of " + jar.packageName + ": " + jar.filePath)
				os.Remove(jar.filePath)
				count++
			}
		}
	}
	log.Println("Total files removed: " + fmt.Sprint(count))
}

func getJarProperties(filePath string) JarProperties {

	archive, err := zip.OpenReader(filePath)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		fileName := filepath.Base(f.Name)

		if !(strings.Compare(f.Name, "META-INF/MANIFEST.MF") == 0 || strings.Compare(fileName, "pom.properties") == 0) {
			continue
		}
		//log.Println("unzipping file ", fileName)

		dstFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			log.Fatal(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			log.Fatal(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			log.Fatal(err)
		}

		dstFile.Close()
		fileInArchive.Close()

		b, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Print(err)
		}

		err = os.Remove(fileName)
		if err != nil {
			log.Fatal(err)
		}

		// try manifest first
		text := string(b)
		jar1 := parseManifest(filePath, text)
		if jar1.packageName != "" {
			return jar1
		}
		jar2 := parsePOM(filePath, text)
		if jar2.packageName != "" {
			return jar2
		}
	}
	log.Println("Failed to parse " + filePath)

	return JarProperties{filePath: ""}
}

func parseManifest(filePath string, text string) JarProperties {
	lines := strings.Split(text, "\n")
	jarProp := JarProperties{filePath: filePath, packageName: ""}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		pair := strings.Split(line, ": ")
		if pair[0] == "Bundle-SymbolicName" {
			jarProp.packageName = pair[1]
		} else if pair[0] == "Bundle-Version" {
			jarProp.version = pair[1]
			// FIXME: Use smart conversion
			jarProp.versionNumber, _ = strconv.Atoi(strings.ReplaceAll(jarProp.version, ".", ""))
		} else if pair[0] == "Bundle-Vendor" {
			jarProp.vendor = pair[1]
		} else if pair[0] == "Bundle-License" {
			jarProp.license = pair[1]
		} else if pair[0] == "Bundle-Name" {
			jarProp.name = pair[1]
		}
	}
	return jarProp
}

func parsePOM(filePath string, text string) JarProperties {
	lines := strings.Split(text, "\n")
	jarProp := JarProperties{filePath: filePath, packageName: ""}
	groupId := ""
	artifactId := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		pair := strings.Split(line, "=")
		if pair[0] == "groupId" {
			groupId = pair[1]
		} else if pair[0] == "artifactId" {
			artifactId = pair[1]
		} else if pair[0] == "version" {
			jarProp.version = pair[1]
			// FIXME: Use smart conversion
			jarProp.versionNumber, _ = strconv.Atoi(strings.ReplaceAll(jarProp.version, ".", ""))
		}
	}
	if groupId != "" && artifactId != "" {
		jarProp.packageName = groupId + "." + artifactId
	}
	return jarProp
}

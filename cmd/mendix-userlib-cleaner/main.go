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
	groupId       string
	artifactId    string
	version       string
	versionNumber int
	filePath      string
	packageName   string
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
		log.Println("Checking " + jar1.filePath)
		if _, ok := keepJars[jar1.packageName]; !ok {
			keepJars[jar1.packageName] = jar1
		}

		// find latest
		for _, jar2 := range jars {
			latestJar := keepJars[jar1.packageName]
			if strings.Compare(latestJar.filePath, jar2.filePath) == 0 {
				// skip self
				continue
			}
			if strings.Compare(latestJar.packageName, jar2.packageName) == 0 {

				// same version?
				// prefer good naming convention when the same version
				goodFileSuffix := jar2.version + ".jar"
				if latestJar.versionNumber == jar2.versionNumber && strings.HasSuffix(jar2.filePath, goodFileSuffix) {
					keepJars[latestJar.packageName] = jar2
				}
				// newer version found
				if latestJar.versionNumber < jar2.versionNumber {
					keepJars[latestJar.packageName] = jar2
				}
			}
		}
	}

	count := 0
	for _, jar := range jars {
		jarToKeep := keepJars[jar.packageName]
		if strings.Compare(jar.filePath, jarToKeep.filePath) != 0 {
			if _, err := os.Stat(jar.filePath); err == nil {
				log.Println(" >> Removing " + jar.packageName + " >> " + jar.filePath)
				//os.Remove(jar.filePath)
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

		if strings.Compare(f.Name, "META-INF/MANIFEST.MF") != 0 {
			continue
		}
		log.Println("unzipping file ", fileName)

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
		pomString := string(b)
		pomTokens := strings.Split(pomString, "\n")
		jarProp := JarProperties{filePath: filePath}
		for _, line := range pomTokens {
			pair := strings.Split(line, "=")
			if pair[0] == "groupId" {
				jarProp.groupId = pair[1]
			} else if pair[0] == "artifactId" {
				jarProp.artifactId = pair[1]
			} else if pair[0] == "version" {
				jarProp.version = pair[1]
				// FIXME: Use smart conversion
				jarProp.versionNumber, _ = strconv.Atoi(strings.ReplaceAll(jarProp.version, ".", ""))
			}
		}
		jarProp.packageName = jarProp.groupId + "." + jarProp.artifactId
		return jarProp

	}
	log.Println("Failed to parse " + filePath)

	return JarProperties{filePath: ""}
}

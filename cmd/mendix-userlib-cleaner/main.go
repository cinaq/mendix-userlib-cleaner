package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"archive/zip"

	"flag"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/op/go-logging"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("main")

var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

type JarProperties struct {
	version       string
	versionNumber int
	filePath      string
	fileName      string
	packageName   string
	name          string
	vendor        string
	license       string
}

func main() {

	flag.String("target", ".", "Path to userlib.")
	flag.Bool("clean", false, "Turn on to actually remove the duplicate JARs.")
	flag.Bool("verbose", false, "Turn on to see debug information.")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	targetDir := viper.GetString("target")
	clean := viper.GetBool("clean")
	verbose := viper.GetBool("verbose")

	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)

	// Set the backends to be used.
	logging.SetBackend(backendFormatter)
	if verbose {
		logging.SetLevel(logging.DEBUG, "main")
	} else {
		logging.SetLevel(logging.INFO, "main")
	}

	jars := listAllJars(targetDir)
	keepJars := computeJarsToKeep(jars)
	count := cleanJars(clean, jars, keepJars)

	if clean {
		log.Infof("Total files removed: %d", count)
	} else {
		log.Infof("Would have removed: %d files", count)
		log.Infof("Use --clean to actually remove above file(s)")
	}

}

func listAllJars(targetDir string) []JarProperties {
	log.Info("Finding and parsing JARs")
	files, err := ioutil.ReadDir(targetDir)
	if err != nil {
		log.Fatal(err)
	}
	jars := []JarProperties{}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".jar") {
			log.Debugf("Processing JAR: %v", f.Name())
			filePath := filepath.Join(targetDir, f.Name())
			jarProp := getJarProps(filePath)
			if strings.Compare(jarProp.filePath, "") != 0 {
				jars = append(jars, jarProp)
			}
		}
	}
	return jars
}

func getJarProps(filePath string) JarProperties {

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

		file, err := ioutil.TempFile("", "jar")
		if err != nil {
			log.Fatal(err)
		}
		defer os.Remove(file.Name())

		dstFile, err := os.OpenFile(file.Name(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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

		b, err := ioutil.ReadFile(file.Name())
		if err != nil {
			log.Warningf("Unable to read file: %v", err)
		}

		// err = os.Remove(file.Name())
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// try manifest first
		text := string(b)
		jar1 := parseManifest(filePath, text)
		if jar1.packageName != "" {
			log.Debugf("Parsed properties from MANIFEST: %v", jar1)
			return jar1
		}
		jar2 := parsePOM(filePath, text)
		if jar2.packageName != "" {
			log.Debugf("Parsed properties from POM: %v", jar2)
			return jar2
		}
	}
	log.Warningf("Failed to parse %v", filePath)

	return JarProperties{filePath: ""}
}

func parseManifest(filePath string, text string) JarProperties {
	lines := strings.Split(text, "\n")
	jarProp := JarProperties{filePath: filePath, packageName: "", fileName: filepath.Base(filePath)}
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
	jarProp := JarProperties{filePath: filePath, packageName: "", fileName: filepath.Base(filePath)}
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

func computeJarsToKeep(jars []JarProperties) map[string]JarProperties {
	log.Info("Computing duplicates")
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
					log.Infof("Preferring file %v over %v", jar2.fileName, latestJar.fileName)
					keepJars[packageName] = jar2
				} else if latestJar.versionNumber < jar2.versionNumber {
					log.Infof("Found newer %v over %v", jar2.fileName, latestJar.fileName)
					keepJars[packageName] = jar2
				}
			}
		}
	}
	return keepJars
}

func cleanJars(remove bool, jars []JarProperties, keepJars map[string]JarProperties) int {
	log.Info("Cleaning...")
	count := 0
	for _, jar := range jars {
		jarToKeep := keepJars[jar.packageName]
		if strings.Compare(jar.filePath, jarToKeep.filePath) != 0 {
			if _, err := os.Stat(jar.filePath); err == nil {
				if remove {
					log.Warningf("Removing duplicate of %v: %v", jar.packageName, jar.fileName)
					os.Remove(jar.filePath)
				} else {
					log.Warningf("Would remove duplicate of %v: %v", jar.packageName, jar.fileName)
				}
				count++
			}
		} else {
			log.Debugf("Keeping jar: %v", jar)
		}
	}
	return count
}

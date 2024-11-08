package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
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
	flag.String("mode", "auto", "Jar parsing mode. Supported options: auto, strict or path to m2ee-log.txt")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	targetDir := viper.GetString("target")
	mode := viper.GetString("mode")
	clean := viper.GetBool("clean")
	verbose := viper.GetBool("verbose")
	regularModes := []string{"auto", "strict"}

	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)

	// Set the backends to be used.
	logging.SetBackend(backendFormatter)
	if verbose {
		logging.SetLevel(logging.DEBUG, "main")
	} else {
		logging.SetLevel(logging.INFO, "main")
	}

	filePaths := listAllFiles(targetDir)
	jars := listAllJars(filePaths, mode)
	keepJars := make(map[string]JarProperties)

	if contains(regularModes, mode) {
		log.Infof("Mode: %v", mode)
		keepJars = computeJarsToKeep(jars)
	} else {
		log.Infof("Mode: m2ee-log at %v", mode)
		keepJars = computeJarsToKeepFromM2eeLog(jars, mode)
	}
	count := cleanJars(clean, filePaths, jars, keepJars)

	if clean {
		log.Infof("Total files removed: %d", count)
	} else {
		log.Infof("Would have removed: %d files", count)
		log.Infof("Use --clean to actually remove above file(s)")
	}

}

func listAllFiles(targetDir string) []string {
	log.Infof("Listing all files in target directory: %v", targetDir)
	files, err := ioutil.ReadDir(targetDir)
	if err != nil {
		log.Fatal(err)
	}
	filePaths := []string{}
	for _, f := range files {
		if !f.IsDir() {
			filePath := filepath.Join(targetDir, f.Name())
			filePaths = append(filePaths, filePath)
		}
	}
	return filePaths
}

func listAllJars(filePaths []string, mode string) []JarProperties {
	log.Info("Finding and parsing JARs")
	jars := []JarProperties{}
	for _, f := range filePaths {
		if strings.HasSuffix(f, ".jar") {
			log.Debugf("Processing JAR: %v", f)
			jarProp := getJarProps(f, mode)
			if strings.Compare(jarProp.filePath, "") != 0 {
				jars = append(jars, jarProp)
			}
		}
	}
	return jars
}

func getJarProps(filePath string, mode string) JarProperties {

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

	jar3 := parseFileName(filePath)
	if jar3.packageName != "" {
		log.Debugf("Parsed properties optimistically: %v", jar3)
		return jar3
	}

	if mode == "auto" {
		jar4 := parseOptimistic(filePath)
		if jar4.packageName != "" {
			log.Debugf("Parsed properties optimistically: %v", jar4)
			return jar4
		}
	}

	log.Warningf("Failed to parse metadata from %v", filePath)

	return JarProperties{filePath: filePath, packageName: filePath, fileName: filepath.Base(filePath), version: ""}
}

func parseManifest(filePath string, text string) JarProperties {
	lines := strings.Split(text, "\n")
	jarProp := JarProperties{filePath: filePath, packageName: "", fileName: filepath.Base(filePath), version: ""}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		pair := strings.Split(line, ": ")

		if len(pair) < 2 {
			continue
		}

		key := pair[0]
		value := pair[1]
		// Automatic-Module-Name - used in org.apache.httpcomponents.httpclient / org.apache.httpcomponents.client5.httpclient5
		if key == "Bundle-SymbolicName" || key == "Extension-Name" || key == "Automatic-Module-Name" {
			jarProp.packageName = value
		} else if key == "Bundle-Version" || key == "Implementation-Version" {
			jarProp.version = value
			jarProp.versionNumber = convertVersionToNumber(jarProp.version)
		} else if key == "Bundle-Vendor" || key == "Implementation-Vendor" {
			jarProp.vendor = value
		} else if key == "Bundle-License" {
			jarProp.license = value
		} else if key == "Bundle-Name" || key == "Implementation-Title" {
			if value == "Apache POI" {
				// skip this because it's a false positive
				continue
			}
			jarProp.name = value
			if jarProp.packageName == "" {
				// only use Bundle-Name as packageName if no alternative exists
				jarProp.packageName = value
			}
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
			jarProp.versionNumber = convertVersionToNumber(jarProp.version)
		}
	}
	if groupId != "" && artifactId != "" {
		jarProp.packageName = groupId + "." + artifactId
	}
	return jarProp
}

func parseOptimistic(filePath string) JarProperties {
	// filePath = junit-4.11.jar
	jarProp := JarProperties{filePath: filePath, packageName: "", fileName: filepath.Base(filePath)}

	// version
	tokens := strings.Split(filePath, "-")
	if len(tokens) > 1 {
		jarProp.version = strings.Replace(tokens[len(tokens)-1], ".jar", "", 1)
		jarProp.versionNumber = convertVersionToNumber(jarProp.version)
	}

	archive, err := zip.OpenReader(filePath)
	if err != nil {
		panic(err)
	}
	defer archive.Close()
	re := regexp.MustCompile(`(org|com)/.*\.class$`)

	for _, f := range archive.File {
		if match := re.MatchString(f.Name); match {
			tokens = strings.Split(f.Name, "/")
			if len(tokens) > 4 {
				// eg. org/example/hello/there/MyClass.class
				tokens = tokens[:4]
			} else if len(tokens) > 3 {
				// eg. org/example/hello/MyClass.class
				tokens = tokens[:3]
			} else if len(tokens) > 2 {
				// eg. org/example/MyClass.class
				tokens = tokens[:2]
			} else {
				tokens = tokens[:1]
			}
			jarProp.packageName = strings.Join(tokens, ".")
			break
		}
	}
	return jarProp
}

func parseFileName(filePath string) JarProperties {
	// Initialize empty JarProperties with filepath and filename
	jarProp := JarProperties{
		filePath:    filePath,
		fileName:    filepath.Base(filePath),
		packageName: "",
	}

	// Split filename on - to get name and version parts
	// e.g. "eventTrackingLibrary-1.0.2.jar" -> ["eventTrackingLibrary", "1.0.2.jar"]
	parts := strings.Split(filepath.Base(filePath), "-")
	if len(parts) < 2 {
		return jarProp
	}

	// Get the version by removing .jar from last part
	version := strings.TrimSuffix(parts[len(parts)-1], ".jar")
	jarProp.version = version
	jarProp.versionNumber = convertVersionToNumber(version)

	// Join all parts except last one to get name
	name := strings.Join(parts[:len(parts)-1], "-")
	jarProp.name = name

	// Convert name to package format (assuming Java package naming convention)
	// e.g. "eventTrackingLibrary" -> "local.eventtracking"
	jarProp.packageName = "local." + strings.ToLower(name)

	return jarProp
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func computeJarsToKeepFromM2eeLog(jars []JarProperties, m2eeLog string) map[string]JarProperties {
	log.Info("Computing evicted jars from m2ee log")
	keepJars := make(map[string]JarProperties)
	evictedJars := getJarFileNames(m2eeLog)

	for _, jar1 := range jars {
		if contains(evictedJars, jar1.fileName) {
			log.Infof("According to m2ee %v was evicted", jar1.fileName)
			continue
		}

		if _, ok := keepJars[jar1.packageName]; !ok {
			keepJars[jar1.packageName] = jar1
		}
	}
	return keepJars
}

func getJarFileNames(m2eeLog string) []string {
	b, err := ioutil.ReadFile(m2eeLog)
	names := []string{}
	if err != nil {
		log.Warningf("Unable to read m2ee-log file: %v", err)
	}
	text := string(b)

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Evicted ") {
			continue
		}

		pair := strings.Split(line, " by ")
		if len(pair) < 2 {
			continue
		}

		fullPath := strings.Replace(pair[0], "Evicted ", "", 1)
		forwardTokens := strings.Split(fullPath, "/")
		forwardLast := forwardTokens[len(forwardTokens)-1]
		backwardTokens := strings.Split(forwardLast, "\\")
		backwardLast := backwardTokens[len(backwardTokens)-1]
		names = append(names, backwardLast)
	}
	log.Debugf("Parsed evicted filenames: %v", names)
	return names
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

func cleanJars(remove bool, filePaths []string, jars []JarProperties, keepJars map[string]JarProperties) int {
	log.Info("Cleaning...")
	jarsCount := 0
	metafilesCount := 0
	for _, jar := range jars {
		jarToKeep := keepJars[jar.packageName]
		if strings.Compare(jar.filePath, jarToKeep.filePath) != 0 {
			for _, filePath := range filePaths {
				if _, err := os.Stat(filePath); err == nil {
					if strings.HasPrefix(filePath, jar.filePath) {
						if remove {
							log.Warningf("Removing file %v: %v", jar.packageName, filePath)
							os.Remove(filePath)
						} else {
							log.Warningf("Would remove file %v: %v", jar.packageName, filePath)
						}
						if strings.HasSuffix(filePath, ".jar") {
							jarsCount++
						} else {
							metafilesCount++
						}
					}
				}
			}
		} else {
			log.Debugf("Keeping jar: %v", jar)
		}
	}
	log.Infof("Clean up %v jars and %v meta files", jarsCount, metafilesCount)
	return jarsCount + metafilesCount
}

func convertVersionToNumber(version string) int {
	// Split version into numeric components
	re := regexp.MustCompile("[0-9]+")
	parts := re.FindAllString(version, -1)

	// Pad with zeros to handle versions with different component counts
	for len(parts) < 4 {
		parts = append(parts, "0")
	}

	// Use decreasing multipliers to ensure proper version ordering
	// This allows for up to 999 in each component
	multipliers := []int{1000000000, 1000000, 1000, 1}
	number := 0

	for i, part := range parts[:4] { // Only use first 4 components
		val, _ := strconv.Atoi(part)
		// Clamp values to avoid overflow
		if val > 999 {
			val = 999
		}
		number += val * multipliers[i]
	}

	return number
}

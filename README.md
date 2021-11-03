# Mendix Userlib Cleaner

This little utility can be used to identify and clean duplicate JARs. It was created mainly for [Mendix](https://mendix.com) apps due to lack of formal dependency management support. This is by no means limited to Mendix use-case. It works for arbitary systems.

Please note that this is not a magic tool that solves all JAR problems. After running the clean tool, always clean your project, rebuild and run it locally.

## In rush?

- Download the windows build from [release](https://github.com/cinaq/mendix-userlib-cleaner/releases)
- Unzip the `mendix-userlib-cleaner.exe` into your userlib you want to clean
- In Explorer window location write `cmd` and press enter. This opens a black terminal window
- In this terminal window write: `.\mendix-userlib-cleaner.exe`
- It should inform you what it will remove. Verify the output and to actually remove the jars write: `.\mendix-userlib-cleaner.exe --clean`

## Why clean userlib?

Most Mendix modules include one or more JAR files. JAR files are java libraries. This is a common approach to extend functionalities in Mendix apps. However as your application ages and grows there's a need to add more modules or update existing ones which introduces newer versions. These JAR's are not managed by Mendix Studio. Due to these dynamics, newer JAR can be introduced at later stage. This causes duplication. Due to JAR duplication it often causes compatibility issues. If you are lucky these errors occur during compile time. At other times these occur during runtime.

## How does this work?

It works as follow:

- list all the JAR files
- for each JAR file compute its properties by extracting `MANIFEST.MF` or `pom.properties` from the JAR file
- Next we loop over the metadata and determine which JAR to keep and discard duplicates. This is done based on the package name (e.g. org.package.velocity) and the version (e.g. 1.7)
- Those marked to be discarded are then removed.

## Usage

```bash
mendix-userlib-cleaner --help
Usage of mendix-userlib-cleaner:
      --clean           Turn on to actually remove the duplicate jars. Default: false
      --mode string     Jar parsing mode. Supported options: auto, strict (default "auto")
      --target string   Path to userlib. Default: . (current directory) (default ".")
      --verbose         Turn on to see debug information. Default: false
pflag: help requested


$ mendix-userlib-cleaner --target ~/resources/jars
01:06:03.237 listAllFiles ▶ INFO 001 Listing all files in target directory: ./resources/jars
01:06:03.237 listAllJars ▶ INFO 002 Finding and parsing JARs
01:06:03.237 listAllJars ▶ DEBU 003 Processing JAR: resources/jars/activation-1.1.1.jar
01:06:03.238 getJarProps ▶ DEBU 004 Parsed properties from MANIFEST: {1.1.1 1001001 resources/jars/activation-1.1.1.jar activation-1.1.1.jar Sun Java System Application Server Sun Java System Application Server Sun Microsystems, Inc. }
01:06:03.238 listAllJars ▶ DEBU 005 Processing JAR: resources/jars/checker-qual-2.5.2.jar
01:06:03.241 getJarProps ▶ DEBU 006 Parsed properties optimistically: {2.5.2 2005002 resources/jars/checker-qual-2.5.2.jar checker-qual-2.5.2.jar org.checkerframework.dataflow   }
01:06:03.241 listAllJars ▶ DEBU 007 Processing JAR: resources/jars/checker-qual-2.5.3.jar
01:06:03.243 getJarProps ▶ DEBU 008 Parsed properties optimistically: {2.5.3 2005003 resources/jars/checker-qual-2.5.3.jar checker-qual-2.5.3.jar org.checkerframework.dataflow   }
01:06:03.243 listAllJars ▶ DEBU 009 Processing JAR: resources/jars/junit-4.11.jar
01:06:03.252 getJarProps ▶ DEBU 00a Parsed properties optimistically: {4.11 4011 resources/jars/junit-4.11.jar junit-4.11.jar org.junit   }
01:06:03.252 listAllJars ▶ DEBU 00b Processing JAR: resources/jars/kafka-streams-2.4.0.jar
01:06:03.257 getJarProps ▶ DEBU 00c Parsed properties optimistically: {2.4.0 2004000 resources/jars/kafka-streams-2.4.0.jar kafka-streams-2.4.0.jar org.apache.kafka   }
01:06:03.257 listAllJars ▶ DEBU 00d Processing JAR: resources/jars/xmlbeans-3.1.0.jar
01:06:03.263 getJarProps ▶ DEBU 00e Parsed properties from MANIFEST: {3.1.0 3001000 resources/jars/xmlbeans-3.1.0.jar xmlbeans-3.1.0.jar org.apache.xmlbeans org.apache.xmlbeans Apache Software Foundation }
01:06:03.263 computeJarsToKeep ▶ INFO 00f Computing duplicates
01:06:03.263 computeJarsToKeep ▶ INFO 010 Found newer checker-qual-2.5.3.jar over checker-qual-2.5.2.jar
01:06:03.263 cleanJars ▶ INFO 011 Cleaning...
01:06:03.263 cleanJars ▶ DEBU 012 Keeping jar: {1.1.1 1001001 resources/jars/activation-1.1.1.jar activation-1.1.1.jar Sun Java System Application Server Sun Java System Application Server Sun Microsystems, Inc. }
01:06:03.263 cleanJars ▶ WARN 013 Would remove file org.checkerframework.dataflow: resources/jars/checker-qual-2.5.2.jar
01:06:03.263 cleanJars ▶ WARN 014 Would remove file org.checkerframework.dataflow: resources/jars/checker-qual-2.5.2.jar.meta
01:06:03.263 cleanJars ▶ DEBU 015 Keeping jar: {2.5.3 2005003 resources/jars/checker-qual-2.5.3.jar checker-qual-2.5.3.jar org.checkerframework.dataflow   }
01:06:03.263 cleanJars ▶ DEBU 016 Keeping jar: {4.11 4011 resources/jars/junit-4.11.jar junit-4.11.jar org.junit   }
01:06:03.263 cleanJars ▶ DEBU 017 Keeping jar: {2.4.0 2004000 resources/jars/kafka-streams-2.4.0.jar kafka-streams-2.4.0.jar org.apache.kafka   }
01:06:03.263 cleanJars ▶ DEBU 018 Keeping jar: {3.1.0 3001000 resources/jars/xmlbeans-3.1.0.jar xmlbeans-3.1.0.jar org.apache.xmlbeans org.apache.xmlbeans Apache Software Foundation }
01:06:03.263 cleanJars ▶ INFO 019 Clean up 1 jars and 1 meta files
01:06:03.263 main ▶ INFO 01a Would have removed: 2 files
01:06:03.263 main ▶ INFO 01b Use --clean to actually remove above file(s)
```

## Extracting metadata

### jar format 1

```
$ cat META-INF/maven/com.sun.istack/istack-commons-runtime/pom.properties
#Created by Apache Maven 3.5.4
groupId=com.sun.istack
artifactId=istack-commons-runtime
version=3.0.8
```

### jar format 2

```
$ cat META-INF/MANIFEST.MF
Manifest-Version: 1.0
Ant-Version: Apache Ant 1.7.0
Created-By: Apache Ant
Package: org.apache.velocity
Build-Jdk: 1.4.2_16
Extension-Name: velocity
Specification-Title: Velocity is a Java-based template engine
Specification-Vendor: Apache Software Foundation
Implementation-Title: org.apache.velocity
Implementation-Vendor-Id: org.apache
Implementation-Vendor: Apache Software Foundation
Implementation-Version: 1.7
Bundle-ManifestVersion: 2
Bundle-Name: Apache Velocity
Bundle-Vendor: Apache Software Foundation
Bundle-SymbolicName: org.apache.velocity
Bundle-Version: 1.7
```

### jar format 3

```
$ cat META-INF/MANIFEST.MF
Manifest-Version: 1.0
Export-Package: com.google.gson;version=2.2.4, com.google.gson.annotat
 ions;version=2.2.4, com.google.gson.reflect;version=2.2.4, com.google
 .gson.stream;version=2.2.4, com.google.gson.internal;version=2.2.4, c
 om.google.gson.internal.bind;version=2.2.4
Bundle-ClassPath: .
Built-By: inder
Bundle-Name: Gson
Created-By: Apache Maven 3.0.4
Bundle-RequiredExecutionEnvironment: J2SE-1.5
Bundle-Vendor: Google Gson Project
Bundle-ContactAddress: http://code.google.com/p/google-gson/
Bundle-Version: 2.2.4
Build-Jdk: 1.7.0_21
Bundle-ManifestVersion: 2
Bundle-Description: Google Gson library
Bundle-SymbolicName: com.google.gson
Archiver-Version: Plexus Archiver
```

### Optimistic parsing

Sometimes there is no usable metadata included in the package. In that case we try to compose the package name by finding the first class file in the jar of which resides in the directory `org` or `com`. e.g. `org.junit`



## License

See the [LICENSE](LICENSE.md) file for license rights and limitations (MIT).

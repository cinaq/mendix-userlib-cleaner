# Mendix Userlib cleaner

This little utility can be used to identify and clean duplicate JARs. It was created mainly for [Mendix](https://mendix.com) apps due to lack of formal dependency management support. This is by no means limited to Mendix use-case. It works for arbitary systems.

Please note that this is not a magic tool that solves all JAR problems. After running the clean tool, always clean your project, rebuild and run it locally.

## Why clean userlib?

Most Mendix modules includes one or more JAR files. JAR files are java libraries. This is a common approach to extend functionalities in Mendix apps. However as your application ages and grows there's a need to add more modules or update existing ones which introduces newer versions. These JAR's are not managed by Mendix Studio. Due to these dynamics, newer JAR can be introduced at later stage. This causes duplication. Due to JAR duplication it often causes compatibility issues. If you are lucky these errors occur during compile time. At other times these occur during runtime.

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
      --target string   Path to userlib. Default: . (current directory) (default ".")
      --verbose         Turn on to see debug information. Default: false
pflag: help requested


$ mendix-userlib-cleaner --target ~/Downloads/userlib        
10:38:58.906 listAllJars ▶ INFO 001 Finding and parsing JARs
10:38:59.525 getJarProps ▶ WARN 002 Failed to parse /Users/xcheng/Downloads/userlib/xercesImpl-2.12.1-sp1.jar
10:38:59.556 computeJarsToKeep ▶ INFO 003 Computing duplicates
10:38:59.556 computeJarsToKeep ▶ INFO 004 Preferring file xmlsec-2.1.4.jar over xmlsec-2.1.4-copy.jar
10:38:59.556 cleanJars ▶ INFO 005 Cleaning...
10:38:59.556 cleanJars ▶ WARN 006 Would remove duplicate of org.apache.santuario.xmlsec: xmlsec-2.1.4-copy.jar
10:38:59.556 main ▶ INFO 007 Would have removed: 1 files
10:38:59.556 main ▶ INFO 008 Use --clean to actually remove above file(s)
```

## Extracting metadata

jar format 1:
```
$ cat META-INF/maven/com.sun.istack/istack-commons-runtime/pom.properties
#Created by Apache Maven 3.5.4
groupId=com.sun.istack
artifactId=istack-commons-runtime
version=3.0.8
```

jar format 2:
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

jar format 3:
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


## License

See the [LICENSE](LICENSE.md) file for license rights and limitations (MIT).
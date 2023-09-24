package util

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var NON_CODE_DIRECTORIES = []string{
	// build directories
	"build", "dist", "bin", "obj", "out", "target", "node_modules", "vendor", "buildout", "buildroot", "build-tools", "build-utils", "builds",
	// IDE directories
	".vscode", ".idea", ".vs",
	// Git and CI/CD directories
	".github", ".gitlab", ".circleci", ".travis", ".gitlab-ci",
	// version control directories
	".svn", ".hg", ".bzr", ".cvs", ".git",
	// environment directories
	".env", ".env.example", ".env.local", ".env.development", ".env.test", ".env.production", ".env.staging",
}

func isNonCodeDirectory(dir string) bool {
	for _, nonCodeDir := range NON_CODE_DIRECTORIES {
		if dir == nonCodeDir {
			return true
		}
	}
	return false
}

var JS_OR_TS = []string{".js", ".ts"}

func IsExtensionSame(path1 string, path2 string) bool {
	ext1 := filepath.Ext(path1)
	ext2 := filepath.Ext(path2)

	if Contains(JS_OR_TS, ext1) {
		return Contains(JS_OR_TS, ext2)
	}
	return ext1 == ext2
}

// GetFilePaths walks through the directory and returns an array of all file paths
func GetFilePaths(dir string) ([]string, error) {
	var filePaths []string

	visit := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if the directory should be skipped
		if f.IsDir() && isNonCodeDirectory(f.Name()) {
			return filepath.SkipDir
		}

		if !f.IsDir() {
			filePaths = append(filePaths, path)
		}
		return nil
	}

	err := filepath.Walk(dir, visit)
	if err != nil {
		return nil, err
	}

	return filePaths, nil
}

var NON_CODE_EXTENSIONS = []string{
	// Text and Document files
	".txt", ".md", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".csv", ".rtf", ".odt", ".ods", ".odp",
	// Image files
	".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".svg", ".psd", ".ai", ".indd",
	// Archive files
	".zip", ".tar", ".gz", ".rar", ".7z", ".bz2", ".z", ".lz", ".arj", ".cab",
	// Audio files
	".mp3", ".wav", ".ogg", ".flac", ".m4a", ".aac", ".wma",
	".aif", ".iff", ".m3u", ".mid", ".mp3", ".mpa",
	// Video files
	".mp4", ".avi", ".mkv", ".flv", ".mov", ".wmv", ".m4v", ".mpg", ".mpeg",
	".3g2", ".3gp", ".asf", ".avi", ".flv", ".m4v", ".mov", ".mp4", ".mpg", ".rm", ".srt", ".swf", ".vob", ".wmv",
	// Log and data files
	".log", ".sql", ".db", ".sqlite", ".bak",
	// Executable and system files
	".exe", ".bat", ".sh", ".dll", ".so", ".jar", ".bin", ".dmg", ".iso", ".img", ".vhd",
	// Icon and Cursor files
	".ico", ".cur", ".icns",
	// Font files
	".ttf", ".otf", ".woff", ".woff2", ".eot", ".fnt",
	// Configuration files
	".ini", ".cfg", ".conf", ".yaml", ".yml", ".toml", ".xml", ".plist",
	// Database files
	".db", ".sqlite", ".sqlite3", ".sql",
	// Temp files
	".tmp", ".temp", ".swp", ".swo", ".swn", ".swo",
	// Lock files
	".lock", ".lck",
	// Cache files
	".cache", ".cch", ".cche", ".csh", ".gch", ".pch", ".swp", ".swo", ".swn", ".swo",
	// Document files
	".doc", ".docx", ".odt", ".pdf", ".rtf", ".tex", ".txt", ".wpd", ".wps",
	// Data files
	".csv", ".dat", ".ged", ".key", ".keychain", ".pps", ".ppt", ".pptx", ".sdf", ".tar", ".tax2016", ".tax2019", ".vcf", ".xml",
	// Executable files
	".apk", ".app", ".bat", ".cgi", ".com", ".exe", ".gadget", ".jar", ".msi", ".py", ".wsf",
	// Game files
	".dem", ".gam", ".nes", ".rom", ".sav",
	// CAD files
	".dwg", ".dxf",
	// GIS files
	".gpx", ".kml", ".kmz",
	// Email files
	".email", ".eml", ".emlx", ".msg", ".oft", ".ost", ".pst", ".vcf",
	// Log files
	".log", ".tlog", ".json",
}

var NON_CODE_FILES = []string{
	// build files
	"build.gradle", "pom.xml", "build.xml", "Makefile", "CMakeLists.txt", "CMakeCache.txt", "cmake_install.cmake",
	// git files
	".gitignore", ".gitattributes", ".gitmodules", ".gitkeep", ".git",
	// config files
	".eslintrc", ".eslintrc.js", ".eslintrc.json", ".eslintrc.yml", ".eslintrc.yaml", ".eslintignore",
	".prettierrc", ".prettierrc.js", ".prettierrc.json", ".prettierrc.yml", ".prettierrc.yaml", ".prettierignore",
	".stylelintrc", ".stylelintrc.js", ".stylelintrc.json", ".stylelintrc.yml", ".stylelintrc.yaml", ".stylelintignore",
	".babelrc", ".babelrc.js", ".babelrc.json", ".babelrc.yml", ".babelrc.yaml",
	".browserslistrc", ".browserslist", ".browserslistrc.js", ".browserslistrc.json", ".browserslistrc.yml", ".browserslistrc.yaml",
	".editorconfig", ".editorconfig.js", ".editorconfig.json", ".editorconfig.yml", ".editorconfig.yaml",
	".flowconfig", ".flowconfig.js", ".flowconfig.json", ".flowconfig.yml", ".flowconfig.yaml",
	".prettierignore", ".prettierignore.js", ".prettierignore.json", ".prettierignore.yml", ".prettierignore.yaml",
	// lock files
	"yarn.lock", "package-lock.json", "Gemfile.lock", "composer.lock", "Podfile.lock",
	// package manager files
	"package.json", "package-lock.json", "yarn.lock", "Gemfile", "Gemfile.lock", "composer.json", "composer.lock", "requirements.txt", "Pipfile", "Pipfile.lock", "pyproject.toml",
	// CI/CD files
	".travis.yml", ".gitlab-ci.yml", ".circleci.yml", ".github", ".gitlab",
	// IDE files
	".vscode", ".idea", ".vs",
	// environment files
	".env", ".env.example", ".env.local", ".env.development", ".env.test", ".env.production", ".env.staging",
	// documentation files
	"README", "README.md", "README.txt", "README.rst", "README.html", "README.htm", "README.markdown", "README.mkd", "README.mdwn", "README.mdown", "README.textile", "README.rdoc", "README.org", "README.adoc", "README.asciidoc", "README.rdoc", "README.1ST", "README.mdx", "README.gfm",
	// license files
	"LICENSE", "LICENSE.txt", "LICENSE.md", "LICENSE.html", "LICENSE.htm", "LICENSE.markdown", "LICENSE.mkd", "LICENSE.mdwn", "LICENSE.mdown", "LICENSE.textile", "LICENSE.rdoc", "LICENSE.org", "LICENSE.adoc", "LICENSE.asciidoc", "LICENSE.rdoc", "LICENSE.1ST", "LICENSE.mdx", "LICENSE.gfm",
	// changelog files
	"CHANGELOG", "CHANGELOG.txt", "CHANGELOG.md", "CHANGELOG.html", "CHANGELOG.htm", "CHANGELOG.markdown", "CHANGELOG.mkd", "CHANGELOG.mdwn", "CHANGELOG.mdown", "CHANGELOG.textile", "CHANGELOG.rdoc", "CHANGELOG.org", "CHANGELOG.adoc", "CHANGELOG.asciidoc", "CHANGELOG.rdoc", "CHANGELOG.1ST", "CHANGELOG.mdx", "CHANGELOG.gfm",
	// contributing files
	"CONTRIBUTING", "CONTRIBUTING.txt", "CONTRIBUTING.md", "CONTRIBUTING.html", "CONTRIBUTING.htm", "CONTRIBUTING.markdown", "CONTRIBUTING.mkd", "CONTRIBUTING.mdwn", "CONTRIBUTING.mdown", "CONTRIBUTING.textile", "CONTRIBUTING.rdoc", "CONTRIBUTING.org", "CONTRIBUTING.adoc", "CONTRIBUTING.asciidoc", "CONTRIBUTING.rdoc", "CONTRIBUTING.1ST", "CONTRIBUTING.mdx", "CONTRIBUTING.gfm",
}

func RemoveNonCodeFiles(filenames []string) []string {
	var codeFiles []string

	for _, filename := range filenames {
		extension := strings.ToLower(filepath.Ext(filename))

		if !(Contains(NON_CODE_EXTENSIONS, extension) ||
			Contains(NON_CODE_FILES, filename)) {
			codeFiles = append(codeFiles, filename)
		}
	}

	return codeFiles
}

const TEMP_REPO_PREFIX = "tempRepo"

func RemoveTempFilePath(path string) string {
	// Use regular expression to find "tempRepo" followed by any number of digits
	re := regexp.MustCompile(TEMP_REPO_PREFIX + `\d+/`)
	matches := re.FindStringSubmatch(path)

	// Get the matched string
	match := matches[0]

	// Find the index where the matched string ends
	index := strings.Index(path, match) + len(match)

	// Remove everything up to and including the matched string
	newPath := path[index:]
	return newPath
}

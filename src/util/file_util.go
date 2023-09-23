package util

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GetFilePaths walks through the directory and returns an array of all file paths
func GetFilePaths(dir string) ([]string, error) {
	var filePaths []string

	visit := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
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

var _NON_CODE_EXTENSIONS = []string{
	// Text and Document files
	".txt", ".md", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".csv", ".rtf", ".odt", ".ods", ".odp",
	// Image files
	".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".svg", ".psd", ".ai", ".indd",
	// Archive files
	".zip", ".tar", ".gz", ".rar", ".7z", ".bz2", ".z", ".lz", ".arj", ".cab",
	// Audio files
	".mp3", ".wav", ".ogg", ".flac", ".m4a", ".aac", ".wma",
	// Video files
	".mp4", ".avi", ".mkv", ".flv", ".mov", ".wmv", ".m4v", ".mpg", ".mpeg",
	// Log and data files
	".log", ".sql", ".db", ".sqlite", ".bak",
	// Executable and system files
	".exe", ".bat", ".sh", ".dll", ".so", ".jar", ".bin", ".dmg", ".iso", ".img", ".vhd",
	// Icon and Cursor files
	".ico", ".cur", ".icns",
	// Git files
	".gitignore", ".gitattributes", ".gitmodules", ".gitkeep", ".git",
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
	// Other files
	"package.json", "package-lock.json", "yarn.lock", "yarn-error.log", "yarn-debug.log", "yarn-integrity",
}

func RemoveNonCodeFiles(filenames []string) []string {
	var codeFiles []string

	for _, filename := range filenames {
		extension := strings.ToLower(filepath.Ext(filename))

		if !Contains(_NON_CODE_EXTENSIONS, extension) {
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

package common

import "strings"

func JoinURLPath(baseURL string, paths ...string) string {
	baseURL = strings.TrimSuffix(baseURL, "/")
	cleanPaths := make([]string, len(paths))
	for i, p := range paths {
		p = strings.Trim(p, "/")
		cleanPaths[i] = p
	}

	if len(cleanPaths) > 0 {
		return baseURL + "/" + strings.Join(cleanPaths, "/")
	}

	return baseURL
}

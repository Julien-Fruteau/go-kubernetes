package env

import "os"

func GetEnvOrDefault(name, other string) string {
	found, ok := os.LookupEnv(name)
	if !ok {
		return other
	}
	return found
}

// motivation: provide an alias for an env var value (defined in lookupMap) instead of the full value
// fallback to the provided env var value if not found in map, or the default value if the env var is not provided
func LookupEnvOrDefault(lookUpMap map[string]string, name, other string) string {
	defaut := GetEnvOrDefault(name, other)

	if value, ok := lookUpMap[defaut]; ok {
		return value
	}

	return defaut
}

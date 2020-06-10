package main

import (
	"log"
	"os"
	"strconv"
)

/*
func Env(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("failed to look up variable: %s", key)
	}
	return val
}
*/

func EnvDefault(key string, def string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	return val
}

func main() {
	bindHost := EnvDefault("BIND_HOST", ":8000")
	enableWebRaw := EnvDefault("ENABLE_WEB", "true")
	enableWeb, err := strconv.ParseBool(enableWebRaw)
	if err != nil {
		log.Fatalf("failed to parse ENABLE_WEB value: %s", enableWebRaw)
	}

	s := NewServer(ServerConfig{
		BindHost:  bindHost,
		EnableWeb: enableWeb,
	})
	err = s.Run()
	if err != nil {
		log.Fatalf("failed to run server: %s", err)
	}
}

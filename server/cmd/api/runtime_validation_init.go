package main

import (
	"log"

	"sovereignconquest/internal/config"
)

func init() {
	if err := validateRuntimeConfiguration(config.Load()); err != nil {
		log.Fatalf("configuration validation failed: %v", err)
	}
}

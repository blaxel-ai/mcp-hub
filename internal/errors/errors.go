package errors

import "log"

// handleError is a helper function for consistent error handling across commands
func HandleError(operation string, err error) {
	if err != nil {
		log.Fatalf("Failed to %s: %v", operation, err)
	}
}

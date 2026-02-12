package gen_test

import (
	"testing"

	// Import the generated package to verify it exists
	pb "github.com/nickfang/personal-dashboard/services/gen/go/weather-provider/v1"
)

func TestGeneratedCodeExists(t *testing.T) {
	// Initialize a struct to ensure the code is valid
	var _ pb.PressureStat
	t.Log("Generated code exists and is importable!")
}

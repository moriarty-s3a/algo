package algo

import (
	"testing"
	log "github.com/sirupsen/logrus"
	"fmt"
)

func TestCalculateRewards(t *testing.T) {
	fmt.Printf("Testing")
	log.SetLevel(log.DebugLevel)
	result := CalculateRewards("http://algo.work/interview/a")
	log.Infof("The final result is: %f ", result)
	if result != 29.0 {
		t.Errorf("Error in calculation.  found %f, expected 29.0", result)
	}
}

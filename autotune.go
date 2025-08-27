/*
@Author: Lzww
@LastEditTime: 2025-8-27 23:16:06
@Description: Auto-tuning mechanism for SafeUDP protocol performance optimization
@Language: Go 1.23.4
*/

package safeudp

// MaxAutoTuneSamples defines the maximum number of pulse samples that can be stored
// in the circular buffer for auto-tuning analysis. This limits memory usage while
// providing sufficient data for pattern detection.
const MaxAutoTuneSamples = 256

// pulse represents a single data point in the auto-tuning system
// It captures both the boolean state (bit) and the sequence number (seq)
// at a specific point in time for pattern analysis
type pulse struct {
	bit bool   // The boolean state/value at this pulse
	seq uint32 // The sequence number when this pulse was recorded
}

// autoTune manages a circular buffer of pulse samples for performance auto-tuning
// It analyzes patterns in the pulse data to detect periodic behavior that can
// be used to optimize protocol parameters
type autoTune struct {
	pulses [MaxAutoTuneSamples]pulse // Circular buffer storing pulse samples
}

// Sample records a new pulse sample with the given bit value and sequence number
// The sample is only stored if the sequence number falls within the valid range
// of the circular buffer to maintain temporal locality of samples
func (tune *autoTune) Sample(bit bool, seq uint32) {
	// Check if the sequence number is within the valid range for our circular buffer
	// We accept samples that are within MaxAutoTuneSamples distance from the first sample
	if seq >= tune.pulses[0].seq && seq <= tune.pulses[0].seq+MaxAutoTuneSamples {
		// Store the pulse in the circular buffer using modulo arithmetic
		// This ensures we wrap around when the buffer is full
		tune.pulses[seq%MaxAutoTuneSamples] = pulse{bit, seq}
	}
}

// FindPeriod analyzes the pulse samples to find the period length of a specific bit pattern
// It looks for transitions from !bit to bit (left edge) and then from bit to !bit (right edge)
// Returns the period length in samples, or -1 if no valid period is found
func (tune *autoTune) FindPeriod(bit bool) int {
	// Start analysis from the first pulse sample
	lastPulse := tune.pulses[0]
	idx := 1

	// Phase 1: Find the left edge (transition from !bit to bit)
	var leftEdge int
	for ; idx < len(tune.pulses); idx++ {
		// Look for a transition where the previous pulse was !bit and current pulse is bit
		if lastPulse.bit != bit && tune.pulses[idx].bit == bit {
			leftEdge = idx // Record the position of the left edge
			break
		}

		lastPulse = tune.pulses[idx]
	}

	// Phase 2: Find the right edge (transition from bit to !bit)
	var rightEdge int
	lastPulse = tune.pulses[leftEdge] // Start from the left edge
	idx = leftEdge + 1

	for ; idx < len(tune.pulses); idx++ {
		// Verify that sequence numbers are consecutive to ensure data integrity
		if lastPulse.seq+1 == tune.pulses[idx].seq {
			// Look for a transition where the previous pulse was bit and current pulse is !bit
			if lastPulse.bit == bit && tune.pulses[idx].bit != bit {
				rightEdge = idx // Record the position of the right edge
				break
			}
		} else {
			// If sequence numbers are not consecutive, the data is invalid
			return -1
		}

		lastPulse = tune.pulses[idx]
	}

	// Return the period length as the distance between right and left edges
	return rightEdge - leftEdge
}

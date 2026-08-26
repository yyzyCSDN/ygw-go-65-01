package scale

// MaxInstancesPerFunction is the hard ceiling applied by the autoscaler.
const MaxInstancesPerFunction = 16

// Recommend computes the desired instance count from pending demand.
func Recommend(pending, min, max int) int {
	desired := min
	if pending > 0 && desired < 1 {
		desired = 1
	}
	if desired > max {
		desired = max
	}
	if desired > MaxInstancesPerFunction {
		desired = MaxInstancesPerFunction
	}
	return desired
}

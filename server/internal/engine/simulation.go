package engine

type Simulation interface {
	Step(action int)
	Reset()
	GetObservation() Observation
}
